# Copyright 2025 Simon Emms <simon@simonemms.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Iterable, List, TypedDict
from dataclasses import dataclass
import yaml
import os
from temporalio.api.common.v1 import Payload
from temporalio.converter import PayloadCodec
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

metadata_keyid = "encryption-key-id"
metadata_encoding = "encoding"
encoding_type = "binary/encrypted"


@dataclass
class Key(TypedDict):
    id: str
    key: str


class EncryptionCodec(PayloadCodec):
    def __init__(self, keys: List[Key]) -> None:
        super().__init__()

        if len(keys) == 0:
            # @todo(sje): this is probably not a TypeError
            raise TypeError(f'Keys are required for AES encryption')

        self.keys = keys

    async def decode(self, payloads: Iterable[Payload]) -> List[Payload]:
        ret: List[Payload] = []
        for p in payloads:
            if p.metadata.get(metadata_encoding, b"").decode() != encoding_type:
                ret.append(p)
                continue

            key_id = p.metadata.get("encryption-key-id", b"").decode()

            key = None
            for k in self.keys:
                if k.get("id") == key_id:
                    key = k.get("key")

            if key == None:
                raise ValueError(f"Unrecognized key ID {key_id}.")

            encryptor = AESGCM(key.encode())
            ret.append(Payload.FromString(self.__decode(encryptor, p.data)))
        return ret

    async def encode(self, payloads: Iterable[Payload]) -> List[Payload]:
        active_key = self.keys[0]
        encryptor = AESGCM(active_key.get("key").encode())

        return [
            Payload(
                metadata={
                    metadata_encoding: encoding_type.encode(),
                    metadata_keyid: active_key.get("id").encode(),
                },
                data=self.__encode(encryptor, p.SerializeToString()),
            )
            for p in payloads
        ]

    @staticmethod
    def __decode(encryptor: AESGCM, data: bytes) -> bytes:
        return encryptor.decrypt(data[:12], data[12:], None)

    @staticmethod
    def __encode(encryptor: AESGCM, data: bytes) -> bytes:
        nonce = os.urandom(12)
        return nonce + encryptor.encrypt(nonce, data, None)

    @staticmethod
    async def create(keypath: str) -> 'EncryptionCodec':
        keys = List[Key]
        with open(keypath) as f:
            data = yaml.safe_load(f)

        keys: List[Key] = [Key(**item) for item in data]

        return EncryptionCodec(keys)
