import type { Env } from './types';
import tracker from './index';
import { importKey, encrypt } from './crypto';
import { describe, it, expect } from 'vitest';

const testKey = 'MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=';
const adminKey = 'admin-secret';
const testPayload = { r: 'test@example.com', s: 'abc123', t: 1704067200 };

type TestEnv = Omit<Partial<Env>, 'DB'> & {
  DB?: Partial<Env['DB']>;
};

function testEnv(overrides: TestEnv = {}): Env {
  const baseDb = {
    prepare: () => {
      throw new Error('unexpected db call');
    },
    batch: () => {
      throw new Error('unexpected db call');
    },
    exec: () => {
      throw new Error('unexpected db call');
    },
    withSession: () => {
      throw new Error('unexpected db call');
    },
    dump: () => {
      throw new Error('unexpected db call');
    },
  } as D1Database;

  const dbOverrides = overrides.DB;
  const db = dbOverrides ? { ...baseDb, ...dbOverrides } : baseDb;

  return {
    TRACKING_KEY: testKey,
    ADMIN_KEY: adminKey,
    ...overrides,
    DB: db,
  } as Env;
}

describe('index errors', () => {
  it('returns JSON 404 for unknown paths', async () => {
    const resp = await tracker.fetch(new Request('https://example.com/not-found'), testEnv());
    const body = (await resp.json()) as { error?: { code: string; message: string } };

    expect(resp.status).toBe(404);
    expect(body.error).toEqual({ code: 'not_found', message: 'not_found' });
  });

  it('returns JSON 400 for invalid tracking payload', async () => {
    const resp = await tracker.fetch(
      new Request('https://example.com/q/not-a-payload'),
      testEnv(),
    );
    const body = (await resp.json()) as { error?: { code: string } };

    expect(resp.status).toBe(400);
    expect(body.error?.code).toBe('invalid_tracking_id');
  });

  it('returns JSON 401 for unauthorized admin access', async () => {
    const resp = await tracker.fetch(new Request('https://example.com/opens'), testEnv());
    const body = (await resp.json()) as { error?: { code: string } };

    expect(resp.status).toBe(401);
    expect(body.error?.code).toBe('unauthorized');
  });

  it('returns JSON 500 when query handling panics', async () => {
    const key = await importKey(testKey);
    const encrypted = await encrypt(testPayload, key);
    const resp = await tracker.fetch(
      new Request(`https://example.com/q/${encrypted}`),
      testEnv({
        DB: {
          prepare: () => {
            throw new Error('db down');
          },
        },
      }),
    );
    const body = (await resp.json()) as { error?: { code: string } };

    expect(resp.status).toBe(500);
    expect(body.error?.code).toBe('internal_error');
  });
});
