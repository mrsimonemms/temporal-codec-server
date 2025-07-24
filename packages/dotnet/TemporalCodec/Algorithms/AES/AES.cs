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

namespace Dotnet.Codec;

using System;
using System.Security.Cryptography;
using System.Collections.Generic;
using System.Text;
using System.Threading.Tasks;
using Temporalio.Api.Common.V1;
using Temporalio.Converters;
using Google.Protobuf;
using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;

public sealed class AESCodec : IPayloadCodec
{
  private static readonly ByteString EncodingByteString = ByteString.CopyFromUtf8("binary/encrypted");
  private static readonly string EncodingHeader = "encoding";
  private static readonly string EncryptionKeyIDHeader = "encryption-key-id";
  private const int NonceSize = 12;
  private const int TagSize = 16;
  private readonly KeyPair[] keys;

  public AESCodec(KeyPair[] keys)
  {
    this.keys = keys;

    if (this.keys.Length == 0)
    {
      throw new ArgumentException("Keys are required for AES encryption");
    }
  }

  public Task<IReadOnlyCollection<Payload>> DecodeAsync(IReadOnlyCollection<Payload> payloads) =>
        Task.FromResult<IReadOnlyCollection<Payload>>(payloads.Select(p =>
        {
          // Ignore if it doesn't have our expected encoding
          if (p.Metadata.GetValueOrDefault(EncodingHeader) != EncodingByteString)
          {
            return p;
          }

          // Find the key
          var keyID = p.Metadata.GetValueOrDefault(EncryptionKeyIDHeader)?.ToStringUtf8();
          KeyPair key = Array.Find(this.keys, k => k.Id == keyID) ?? throw new InvalidOperationException($"Unrecognized key ID {keyID}");

          // Decrypt
          return Payload.Parser.ParseFrom(Decrypt(Encoding.ASCII.GetBytes(key.Key!), p.Data.ToByteArray()));
        }).ToList());

  public Task<IReadOnlyCollection<Payload>> EncodeAsync(IReadOnlyCollection<Payload> payloads) =>
         Task.FromResult<IReadOnlyCollection<Payload>>(payloads.Select(p =>
         {
           KeyPair key = this.keys[0];

           return new Payload()
           {
             Metadata =
                 {
                    [EncodingHeader] = EncodingByteString,
                    [EncryptionKeyIDHeader] = ByteString.CopyFromUtf8(key.Id),
                 },
             Data = ByteString.CopyFrom(Encrypt(Encoding.ASCII.GetBytes(key.Key!), p.ToByteArray())),
           };
         }).ToList());

  private static byte[] Decrypt(byte[] key, byte[] data)
  {
    var bytes = new byte[data.Length - NonceSize - TagSize];

    using var aes = new AesGcm(key, TagSize);
    aes.Decrypt(
        data.AsSpan(0, NonceSize), data.AsSpan(NonceSize, bytes.Length), data.AsSpan(NonceSize + bytes.Length, TagSize), bytes.AsSpan());
    return bytes;
  }

  private static byte[] Encrypt(byte[] key, byte[] data)
  {
    var bytes = new byte[NonceSize + TagSize + data.Length];

    // Generate random nonce
    var nonceSpan = bytes.AsSpan(0, NonceSize);
    RandomNumberGenerator.Fill(nonceSpan);

    // Perform encryption
    using var aes = new AesGcm(key, TagSize);
    aes.Encrypt(nonceSpan, data, bytes.AsSpan(NonceSize, data.Length), bytes.AsSpan(NonceSize + data.Length, TagSize));
    return bytes;
  }

  public static async Task<AESCodec> Create(string? filepath)
  {
    if (filepath == null)
    {
      throw new ArgumentException("File path not specified");
    }

    using StreamReader reader = new(filepath);
    string yaml = await reader.ReadToEndAsync();

    var deserializer = new DeserializerBuilder()
      .WithNamingConvention(UnderscoredNamingConvention.Instance)
      .Build();

    var keys = deserializer.Deserialize<KeyPair[]>(yaml);

    // Validate the keys
    for (int i = 0; i < keys.Length; i++)
    {
      if (keys[i].Id == "")
      {
        throw new ArgumentException($"Keys \"id\" parameter is required for position {i}");
      }
      if (keys[i].Key == "")
      {
        throw new ArgumentException($"Keys \"key\" parameter is required for position {i}");
      }
    }

    return new AESCodec(keys);
  }
}

public class KeyPair
{
  public string? Id { get; set; }
  public string? Key { get; set; }
}
