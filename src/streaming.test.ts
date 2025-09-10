import { getLiveStreamKey } from './streaming';
import { config } from '@grafana/runtime';

// Mock the @grafana/runtime module
jest.mock('@grafana/runtime', () => ({
  config: {
    bootData: {
      user: {
        orgId: 1,
      },
    },
  },
}));

// Mock crypto.subtle for consistent testing
const mockDigest = jest.fn();
Object.defineProperty(global, 'crypto', {
  value: {
    subtle: {
      digest: mockDigest,
    },
  },
  writable: true,
});

describe('getLiveStreamKey', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset to default orgId
    (config.bootData.user as any).orgId = 1;
  });

  it('should generate a consistent key for the same query', async () => {
    // Mock SHA-1 hash result - first 8 bytes of a known hash
    const mockHashBuffer = new ArrayBuffer(20); // SHA-1 produces 20 bytes
    const mockHashArray = new Uint8Array(mockHashBuffer);
    // Set first 8 bytes to known values for predictable output
    mockHashArray.set([0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    const key1 = await getLiveStreamKey(datasourceUid, topic);
    const key2 = await getLiveStreamKey(datasourceUid, topic);

    expect(key1).toBe(key2);
    expect(key1).toBe('mqtt-datasource-uid/123456789abcdef0/1');
  });

  it('should generate different keys for different topics', async () => {
    const mockHashBuffer1 = new ArrayBuffer(20);
    const mockHashArray1 = new Uint8Array(mockHashBuffer1);
    mockHashArray1.set([0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88]);

    const mockHashBuffer2 = new ArrayBuffer(20);
    const mockHashArray2 = new Uint8Array(mockHashBuffer2);
    mockHashArray2.set([0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00]);

    mockDigest
      .mockResolvedValueOnce(mockHashBuffer1)
      .mockResolvedValueOnce(mockHashBuffer2);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic1 = 'sensor/temperature';
    const topic2 = 'sensor/humidity';

    const key1 = await getLiveStreamKey(datasourceUid, topic1);
    const key2 = await getLiveStreamKey(datasourceUid, topic2);

    expect(key1).not.toBe(key2);
    expect(key1).toBe('mqtt-datasource-uid/1122334455667788/1');
    expect(key2).toBe('mqtt-datasource-uid/99aabbccddeeff00/1');
  });

  it('should include orgId in the key', async () => {
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    mockHashArray.set([0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    // Test with different orgId
    (config.bootData.user as any).orgId = 42;

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    const key = await getLiveStreamKey(datasourceUid, topic);

    expect(key).toBe('mqtt-datasource-uid/123456789abcdef0/42');
  });

  it('should generate different keys for different organizations', async () => {
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    mockHashArray.set([0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    // Test with orgId 1
    (config.bootData.user as any).orgId = 1;
    const key1 = await getLiveStreamKey(datasourceUid, topic);

    // Test with orgId 2
    (config.bootData.user as any).orgId = 2;
    const key2 = await getLiveStreamKey(datasourceUid, topic);

    expect(key1).not.toBe(key2);
    expect(key1).toBe('mqtt-datasource-uid/123456789abcdef0/1');
    expect(key2).toBe('mqtt-datasource-uid/123456789abcdef0/2');
  });

  it('should handle undefined topic', async () => {
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    mockHashArray.set([0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = undefined;

    const key = await getLiveStreamKey(datasourceUid, topic);

    expect(key).toBe('mqtt-datasource-uid/123456789abcdef0/1');
    
    // Verify that JSON.stringify was called with undefined topic
    expect(mockDigest).toHaveBeenCalledWith(
      'SHA-1',
      new TextEncoder().encode(JSON.stringify({ topic: undefined }))
    );
  });

  it('should handle missing datasource uid', async () => {
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    mockHashArray.set([0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = undefined as any;
    const topic = 'sensor/temperature';

    const key = await getLiveStreamKey(datasourceUid, topic);

    expect(key).toBe('undefined/123456789abcdef0/1');
  });

  it('should use SHA-1 hashing correctly', async () => {
    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    await getLiveStreamKey(datasourceUid, topic);

    expect(mockDigest).toHaveBeenCalledWith(
      'SHA-1',
      new TextEncoder().encode(JSON.stringify({ topic: 'sensor/temperature' }))
    );
  });

  it('should only use first 8 bytes of hash', async () => {
    // Create a mock hash buffer with all bytes set to different values
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    for (let i = 0; i < 20; i++) {
      mockHashArray[i] = i + 1; // 1, 2, 3, ..., 20
    }
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    const key = await getLiveStreamKey(datasourceUid, topic);

    // Should only include first 8 bytes (01, 02, 03, 04, 05, 06, 07, 08)
    expect(key).toBe('mqtt-datasource-uid/0102030405060708/1');
  });

  it('should pad hex values with leading zeros', async () => {
    const mockHashBuffer = new ArrayBuffer(20);
    const mockHashArray = new Uint8Array(mockHashBuffer);
    // Set values that would be single digit in hex
    mockHashArray.set([0x01, 0x02, 0x03, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e]);
    mockDigest.mockResolvedValue(mockHashBuffer);

    const datasourceUid = 'mqtt-datasource-uid';
    const topic = 'sensor/temperature';

    const key = await getLiveStreamKey(datasourceUid, topic);

    // Should pad single digit hex values with leading zeros
    expect(key).toBe('mqtt-datasource-uid/0102030a0b0c0d0e/1');
  });
});
