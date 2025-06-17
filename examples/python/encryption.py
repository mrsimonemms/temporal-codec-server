from typing import Iterable, List, TypedDict
from dataclasses import dataclass
import cramjam
import yaml
from temporalio.api.common.v1 import Payload
from temporalio.converter import PayloadCodec

metadata_keyid = "encryption-key-id"
metadata_encoding = "encoding"
encoding_type = b"binary/encrypted"


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
            if p.metadata.get("encoding", b"").decode() != "binary/snappy":
                ret.append(p)
                continue
            ret.append(Payload.FromString(
                bytes(cramjam.snappy.decompress(p.data))))
        return ret

    async def encode(self, payloads: Iterable[Payload]) -> List[Payload]:
        active_key = self.keys[0]
        return [
            Payload(
                metadata={
                    metadata_encoding: encoding_type,
                    metadata_keyid: active_key.get("id"),
                },
                data=(bytes(cramjam.snappy.compress(p.SerializeToString()))),
            )
            for p in payloads
        ]

    @staticmethod
    async def create(keypath: str) -> 'EncryptionCodec':
        keys = List[Key]
        with open(keypath) as f:
            data = yaml.safe_load(f)

        keys: List[Key] = [Key(**item) for item in data]

        return EncryptionCodec(keys)
