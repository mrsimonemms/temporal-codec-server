/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {
  METADATA_ENCODING_KEY,
  Payload,
  PayloadCodec,
  PayloadConverterError,
  ValueError,
} from '@temporalio/common';
import { decode, encode } from '@temporalio/common/lib/encoding';
import { temporal } from '@temporalio/proto';
import { webcrypto as crypto } from 'crypto';
import * as fs from 'fs/promises';
import { parse } from 'yaml';

export interface Key {
  id: string;
  key: string;
}

const ENCODING = 'binary/encrypted';
const METADATA_ENCRYPTION_KEY_ID = 'encryption-key-id';

export class AESCodec implements PayloadCodec {
  constructor(private keys: Key[]) {
    if (keys.length === 0) {
      throw new PayloadConverterError('Keys are required for AES encryption');
    }
  }

  decode(payloads: Payload[]): Promise<Payload[]> {
    return Promise.all(
      payloads.map(async (payload) => {
        if (
          !payload.metadata ||
          decode(payload.metadata[METADATA_ENCODING_KEY]) !== ENCODING
        ) {
          return payload;
        }
        if (!payload.data) {
          throw new ValueError('Payload data is missing');
        }

        const keyIdBytes = payload.metadata[METADATA_ENCRYPTION_KEY_ID];
        if (!keyIdBytes) {
          throw new ValueError(
            'Unable to decrypt Payload without encryption key id',
          );
        }

        const keyId = decode(keyIdBytes);
        const key = await this.fetchKey(keyId);

        const decryptedPayloadBytes = await decrypt(payload.data, key);

        return temporal.api.common.v1.Payload.decode(decryptedPayloadBytes);
      }),
    );
  }

  async encode(payloads: Payload[]): Promise<Payload[]> {
    if (this.keys.length === 0) {
      throw new PayloadConverterError('');
    }
    const keyId = this.keys[0].id;

    return Promise.all(
      payloads.map(async (payload) => ({
        metadata: {
          [METADATA_ENCODING_KEY]: encode(ENCODING),
          [METADATA_ENCRYPTION_KEY_ID]: encode(keyId),
        },
        // Encrypt entire payload, preserving metadata
        data: await encrypt(
          temporal.api.common.v1.Payload.encode(payload).finish(),
          await this.fetchKey(keyId),
        ),
      })),
    );
  }

  static async create(keyPath?: string): Promise<AESCodec> {
    if (!keyPath) {
      throw new PayloadConverterError('Encryption key path is required');
    }

    const data = await fs.readFile(keyPath, 'utf8');

    // Validate the keys
    const keys = (parse(data) as Key[]).map(({ id, key }, position) => {
      if (!id) {
        throw new PayloadConverterError(
          `Keys "id" parameter is required for position ${position}`,
        );
      }
      if (!key) {
        throw new PayloadConverterError(
          `Keys "key" parameter is required for position ${position}`,
        );
      }

      return {
        id,
        key,
      };
    });

    return new AESCodec(keys);
  }

  private fetchKey(keyId: string): Promise<crypto.CryptoKey> {
    const targetKey = this.keys.find(({ id }) => id === keyId);
    if (!targetKey) {
      throw new PayloadConverterError('Unknown key ID');
    }

    return crypto.subtle.importKey(
      'raw',
      Buffer.from(targetKey.key),
      {
        name: 'AES-GCM',
      },
      true,
      ['encrypt', 'decrypt'],
    );
  }
}

const CIPHER = 'AES-GCM';
const IV_LENGTH_BYTES = 12;
const TAG_LENGTH_BYTES = 16;

export async function encrypt(
  data: Uint8Array,
  key: crypto.CryptoKey,
): Promise<Uint8Array> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH_BYTES));
  const encrypted = await crypto.subtle.encrypt(
    {
      name: CIPHER,
      iv,
      tagLength: TAG_LENGTH_BYTES * 8,
    },
    key,
    data,
  );

  return Buffer.concat([iv, new Uint8Array(encrypted)]);
}

export async function decrypt(
  encryptedData: Uint8Array,
  key: crypto.CryptoKey,
): Promise<Uint8Array> {
  const iv = encryptedData.subarray(0, IV_LENGTH_BYTES);
  const ciphertext = encryptedData.subarray(IV_LENGTH_BYTES);
  const decrypted = await crypto.subtle.decrypt(
    {
      name: CIPHER,
      iv,
      tagLength: TAG_LENGTH_BYTES * 8,
    },
    key,
    ciphertext,
  );

  return new Uint8Array(decrypted);
}
