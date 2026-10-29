import { ContactModule } from '../src/contact';
import { NoStore } from '../src/store';
import { ConsoleLogging } from '../src/logging';
import { Card } from '../src/types';
import { waitFor } from '../__mocks__/waitFor';
import { getContactChannels } from '../src/net/getContactChannels';

let notified = 5;
const card = () => ({
  id: 'C000A',
  revision: 2,
  data: {
    detailRevision: 1,
    profileRevision: 1,
    notifiedProfile: 1,
    notifiedArticle: 0,
    notifiedChannel: notified,
    cardDetail: { status: 'connected', statusUpdated: 1, token: 'T', notes: '', groups: [] },
    cardProfile: { guid: 'G', handle: 'H', name: 'N', description: '', location: '', imageSet: false, version: 'V', node: 'URL_A', seal: '', revision: 1 },
  },
});

jest.mock('../src/net/getCards', () => ({ getCards: jest.fn(async () => [card()]) }));
jest.mock('../src/net/getContactChannels', () => ({ getContactChannels: jest.fn() }));
jest.mock('../src/net/fetchUtil', () => ({ fetchWithTimeout: jest.fn(), checkResponse: jest.fn() }));

class MarkerStore extends NoStore {
  public markers = new Map<string, string>();
  public async setMarker(guid: string, type: string, id: string, value: string) { this.markers.set(`${type}/${id}`, value); }
  public async clearMarker(guid: string, type: string, id: string) { this.markers.delete(`${type}/${id}`); }
  public async getMarkers(guid: string, type: string) {
    return [...this.markers.entries()].filter(([k]) => k.startsWith(type + '/')).map(([k, value]) => ({ id: k.slice(type.length + 1), value }));
  }
}

const mockChannels = getContactChannels as jest.Mock;
const log = new ConsoleLogging();
const make = (store: MarkerStore) => new ContactModule(log, store, null as any, null, 'g', 'tok', 'test_url', false, []);

beforeEach(() => {
  mockChannels.mockReset();
  jest.spyOn(console, 'warn').mockImplementation(() => {});
});
afterEach(() => jest.restoreAllMocks());

test('offsync channel sync is retried on next revision and cleared', async () => {
  const store = new MarkerStore();
  mockChannels.mockRejectedValueOnce(new Error('timeout')).mockResolvedValue([]);
  const contact = make(store);
  let cards: Card[] = [];
  contact.addCardListener((c) => { cards = c; });
  contact.setRevision(2);
  await waitFor(() => store.markers.has('offsync_channel_card/C000A'));
  await waitFor(() => cards.length === 1 && cards[0].offsync === true);

  notified = 6;
  contact.setRevision(3);
  await waitFor(() => !store.markers.has('offsync_channel_card/C000A'));
  await waitFor(() => cards[0].offsync === false);
  expect(mockChannels).toHaveBeenCalledTimes(2);
  await contact.close();
});

test('persisted offsync marker is retried after reload and cleared', async () => {
  const store = new MarkerStore();
  store.markers.set('offsync_channel_card/C000A', '5');
  mockChannels.mockResolvedValue([]);
  const first = make(store);
  first.setRevision(2);
  await first.close();
  store.markers.set('offsync_channel_card/C000A', '5');
  mockChannels.mockClear();

  const contact = make(store);
  contact.setRevision(2);
  await waitFor(() => !store.markers.has('offsync_channel_card/C000A'));
  expect(mockChannels).toHaveBeenCalled();
  await contact.close();
});

test('periodic timer retries an offsync contact without new revisions', async () => {
  jest.useFakeTimers({ doNotFake: ['setImmediate', 'nextTick'] });
  try {
    const store = new MarkerStore();
    mockChannels.mockRejectedValueOnce(new Error('timeout')).mockResolvedValue([]);
    const contact = make(store);
    contact.setRevision(2);
    for (let i = 0; i < 50 && !store.markers.has('offsync_channel_card/C000A'); i++) {
      await jest.advanceTimersByTimeAsync(100);
    }
    expect(store.markers.has('offsync_channel_card/C000A')).toBe(true);
    for (let i = 0; i < 20 && store.markers.has('offsync_channel_card/C000A'); i++) {
      await jest.advanceTimersByTimeAsync(10000);
    }
    expect(store.markers.has('offsync_channel_card/C000A')).toBe(false);
    expect(mockChannels).toHaveBeenCalledTimes(2);
    await contact.close();
  } finally {
    jest.useRealTimers();
  }
});
