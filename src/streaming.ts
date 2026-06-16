import { config } from '@grafana/runtime';

/**
 * Calculate a unique key for the query.  The key is used to pick a channel and should
 * be unique for each distinct query execution plan.  This key is not secure and is only picked to avoid
 * possible collisions
 */
export async function getLiveStreamKey(datasourceUid: string, topic?: string): Promise<string> {
  const str = JSON.stringify({ topic });

  const orgId = config.bootData.user.orgId;
  const msgUint8 = new TextEncoder().encode(str); // encode as (utf-8) Uint8Array
  let hashBuffer;
  if (crypto.subtle === undefined) {
    // Fall back to our own sha1 if we don't have crypto.subtle (e.g. not on localhost or over https)
    hashBuffer = sha1(msgUint8);
  }
  else {
    hashBuffer = await crypto.subtle.digest('SHA-1', msgUint8); // hash the message
  }
  const hashArray = Array.from(new Uint8Array(hashBuffer.slice(0, 8))); // first 8 bytes
  return `${datasourceUid}/${hashArray.map((b) => b.toString(16).padStart(2, '0')).join('')}/${orgId}`;
}

function sha1(message: Uint8Array): ArrayBuffer {
  let h0 = 0x67452301;
  let h1 = 0xEFCDAB89;
  let h2 = 0x98BADCFE;
  let h3 = 0x10325476;
  let h4 = 0xC3D2E1F0;

  const message_length = message.length * 8;

  // We need to pad with 0x80, zeroes, and then the message length as a 64-bit integer, to take us up
  // to a multiple of 64 bytes.
  let buf = new Uint8Array(64 * Math.ceil((message.length + 9) / 64));
  buf.set(message);
  buf[message.length] = 0x80;


  // Bitwise operators truncate to 32 bits, we need to explicitly take the high bits
  const message_length_high = Math.floor(message_length / 0x100000000);
  buf[buf.length - 8] = (message_length_high & 0xff000000) >>> 24;
  buf[buf.length - 7] = (message_length_high & 0x00ff0000) >>> 16;
  buf[buf.length - 6] = (message_length_high & 0x0000ff00) >>> 8;
  buf[buf.length - 5] = (message_length_high & 0x000000ff);

  buf[buf.length - 4] = (message_length & 0xff000000) >>> 24;
  buf[buf.length - 3] = (message_length & 0x00ff0000) >>> 16;
  buf[buf.length - 2] = (message_length & 0x0000ff00) >>> 8;
  buf[buf.length - 1] = (message_length & 0x000000ff);

  for (let chunkIdx = 0; chunkIdx < buf.length; chunkIdx += 64) {
    let words = []
    for (let wordIdx = 0; wordIdx < 80; wordIdx += 1) {
      if (wordIdx < 16) {
        words[wordIdx] = buf[chunkIdx + (wordIdx * 4)]     << 24 |
                         buf[chunkIdx + (wordIdx * 4) + 1] << 16 |
                         buf[chunkIdx + (wordIdx * 4) + 2] << 8  |
                         buf[chunkIdx + (wordIdx * 4) + 3];
      } else {
        const withoutRotation: number = words[wordIdx - 3] ^ words[wordIdx - 8] ^ words[wordIdx - 14] ^ words[wordIdx - 16];
        words[wordIdx] = (withoutRotation << 1) | (withoutRotation >>> 31);
      }
    }

    let a = h0;
    let b = h1;
    let c = h2;
    let d = h3;
    let e = h4;

    for (let i = 0; i < 80; i += 1) {
      let f;
      let k;
      if (i < 20) {
        f = (b & c) | ((~b) & d);
        k = 0x5A827999;
      } else if (i < 40) {
        f = b ^ c ^ d;
        k = 0x6ED9EBA1;
      } else if (i < 60) {
        f = (b & c) | (b & d) | (c & d);
        k = 0x8F1BBCDC;
      } else {
        f = b ^ c ^ d;
        k = 0xCA62C1D6;
      }

      const temp = ((a << 5) | (a >>> 27)) + f + e + k + words[i];
      e = d;
      d = c;
      c = (b << 30) | (b >>> 2);
      b = a;
      a = temp;
    }

    h0 += a
    h1 += b
    h2 += c
    h3 += d
    h4 += e
  }

  const retBuffer = new ArrayBuffer(20);
  const view = new DataView(retBuffer);
  view.setUint32(0, h0, false);
  view.setUint32(4, h1, false);
  view.setUint32(8, h2, false);
  view.setUint32(12, h3, false);
  view.setUint32(16, h4, false);
  return retBuffer;
}
