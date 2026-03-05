export interface Env {
  DB: D1Database;
  TRACKING_KEY: string;
  ADMIN_KEY: string;
}

export interface PixelPayload {
  r: string; // recipient
  s: string; // subject hash (first 6 chars)
  t: number; // sent timestamp (unix)
}

export interface OpenRecord {
  id: number;
  tracking_id: string;
  recipient: string;
  subject_hash: string;
  sent_at: string;
  opened_at: string;
  ip: string;
  user_agent: string;
  country: string | null;
  region: string | null;
  city: string | null;
  timezone: string | null;
  is_bot: number;
  bot_type: string | null;
}

/** Cloudflare request.cf geo fields we use (subset of IncomingRequestCfProperties). */
export interface CfGeo {
  country?: string | null;
  region?: string | null;
  city?: string | null;
  timezone?: string | null;
}

/** Row shape from SELECT opened_at, ip, city, region, country, timezone, is_bot, bot_type. */
export interface OpenQueryRow {
  opened_at: string;
  ip: string | null;
  city: string | null;
  region: string | null;
  country: string | null;
  timezone: string | null;
  is_bot: number;
  bot_type: string | null;
}

/** Row shape for admin /opens list (full row). */
export interface OpenAdminRow {
  tracking_id: string;
  recipient: string;
  subject_hash: string;
  sent_at: string;
  opened_at: string;
  is_bot: number;
  bot_type: string | null;
  city: string | null;
  region: string | null;
  country: string | null;
}
