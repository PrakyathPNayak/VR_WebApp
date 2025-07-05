/**
 * Cryptography utilities for secure communication
 */

let aesKey = null;
let iv = null;

function base64Encode(buffer) {
  return btoa(String.fromCharCode(...buffer));
}

function base64Decode(str) {
  return new Uint8Array([...atob(str)].map((c) => c.charCodeAt(0)));
}

async function performKeyExchange(encodedPem) {
  const pem = atob(encodedPem);
  const encrypt = new JSEncrypt();
  encrypt.setPublicKey(pem);

  const aesKeyRaw = crypto.getRandomValues(new Uint8Array(32));
  iv = crypto.getRandomValues(new Uint8Array(12));

  aesKey = await crypto.subtle.importKey(
    "raw",
    aesKeyRaw,
    { name: "AES-GCM" },
    false,
    ["encrypt", "decrypt"],
  );

  // Convert raw bytes to base64 for RSA encryption
  const aesKeyB64 = btoa(String.fromCharCode(...aesKeyRaw));
  const encryptedKey = encrypt.encrypt(aesKeyB64);

  return {
    encrypted_key: encryptedKey,
    iv: base64Encode(iv),
  };
}

async function encryptMessage(messageObj) {
  if (!aesKey || !iv) return null;

  const encoded = new TextEncoder().encode(JSON.stringify(messageObj));
  const nonce = crypto.getRandomValues(new Uint8Array(12));

  try {
    const encrypted = await crypto.subtle.encrypt(
      {
        name: "AES-GCM",
        iv: nonce,
        tagLength: 128,
      },
      aesKey,
      encoded,
    );

    const nonceAndEncrypted = new Uint8Array(
      nonce.byteLength + encrypted.byteLength,
    );
    nonceAndEncrypted.set(nonce, 0);
    nonceAndEncrypted.set(new Uint8Array(encrypted), nonce.byteLength);

    return nonceAndEncrypted.buffer;
  } catch (e) {
    console.error("Failed to encrypt message:", e);
    return null;
  }
}

async function decryptMessage(data) {
  if (!aesKey) return null;

  const nonceSize = 12;
  const tagSize = 16;

  if (data.byteLength < nonceSize + tagSize) return null;

  const nonce = data.slice(0, nonceSize);
  const ciphertext = data.slice(nonceSize, data.byteLength - tagSize);
  const tag = data.slice(data.byteLength - tagSize);

  const fullData = new Uint8Array(ciphertext.byteLength + tag.byteLength);
  fullData.set(new Uint8Array(ciphertext), 0);
  fullData.set(new Uint8Array(tag), ciphertext.byteLength);

  try {
    const decrypted = await crypto.subtle.decrypt(
      {
        name: "AES-GCM",
        iv: nonce,
        tagLength: 128,
      },
      aesKey,
      fullData,
    );

    return JSON.parse(new TextDecoder().decode(decrypted));
  } catch (e) {
    console.warn("Failed to decrypt control message", e);
    return null;
  }
}

function isEncryptionReady() {
  return aesKey !== null && iv !== null;
}
