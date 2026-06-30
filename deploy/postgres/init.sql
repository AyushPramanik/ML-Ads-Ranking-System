-- Schema and seed data for the ML Ads Ranking System.
-- Executed once by the PostgreSQL container on first start.
--
-- The seeded ads/campaigns mirror ranking/internal/store/sample.go so the
-- database-backed and in-memory catalogs behave identically.

BEGIN;

-- --------------------------------------------------------------------------
-- Core entities
-- --------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id         BIGINT PRIMARY KEY,
    age        INTEGER NOT NULL,
    gender     TEXT    NOT NULL,
    country    TEXT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS campaigns (
    id         BIGINT PRIMARY KEY,
    name       TEXT    NOT NULL,
    advertiser TEXT    NOT NULL,
    category   TEXT    NOT NULL,
    budget     NUMERIC(12,2) NOT NULL CHECK (budget >= 0),
    status     TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ads (
    id             BIGINT PRIMARY KEY,
    campaign_id    BIGINT NOT NULL REFERENCES campaigns(id),
    title          TEXT   NOT NULL,
    category       TEXT   NOT NULL,
    historical_ctr NUMERIC(6,5) NOT NULL DEFAULT 0 CHECK (historical_ctr >= 0 AND historical_ctr <= 1),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ads_campaign_id ON ads(campaign_id);

-- Impression and click logs: the raw events an offline pipeline would aggregate
-- into training data. Included to model the full data lifecycle.
CREATE TABLE IF NOT EXISTS impressions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    ad_id       BIGINT NOT NULL REFERENCES ads(id),
    device      TEXT   NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    shown_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_impressions_user_id ON impressions(user_id);
CREATE INDEX IF NOT EXISTS idx_impressions_ad_id ON impressions(ad_id);

CREATE TABLE IF NOT EXISTS clicks (
    id            BIGSERIAL PRIMARY KEY,
    impression_id BIGINT NOT NULL REFERENCES impressions(id),
    clicked_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- --------------------------------------------------------------------------
-- Seed: campaigns
-- --------------------------------------------------------------------------
INSERT INTO campaigns (id, name, advertiser, category, budget, status) VALUES
    (101, 'Spring Retail Push',   'NorthPeak Apparel',   'retail',  5000,  'active'),
    (102, 'Card Acquisition Q3',  'Meridian Bank',       'finance', 12000, 'active'),
    (103, 'Raid Boss UA',         'Pixel Forge Games',   'gaming',  8000,  'active'),
    (104, 'Summer Escapes',       'Voyago Travel',       'travel',  6500,  'active'),
    (105, 'Weeknight Dinners',    'FreshPlate',          'food',    4200,  'active'),
    (106, 'EV Lease Drive',       'Volt Motors',         'auto',    15000, 'active'),
    (107, 'Device Launch',        'Nimbus Electronics',  'tech',    9000,  'active'),
    (108, 'Wellbeing Always-On',  'CalmCare Health',     'health',  7000,  'active'),
    (109, 'Outlet Clearance',     'NorthPeak Apparel',   'retail',  3000,  'active'),
    (110, 'Index Investing',      'Meridian Bank',       'finance', 11000, 'active'),
    (111, 'Console Bundle (EOL)', 'Pixel Forge Games',   'gaming',  2000,  'paused'),
    (112, 'Road Trip Rentals',    'Voyago Travel',       'travel',  5500,  'active')
ON CONFLICT (id) DO NOTHING;

-- --------------------------------------------------------------------------
-- Seed: ads (mirrors store.SampleAds)
-- --------------------------------------------------------------------------
INSERT INTO ads (id, campaign_id, title, category, historical_ctr) VALUES
    (1,  101, 'Summer Sneaker Sale',          'retail',  0.082),
    (2,  101, 'Designer Bags 40% Off',        'retail',  0.061),
    (3,  102, '0% APR Balance Transfer',      'finance', 0.019),
    (4,  102, 'High-Yield Savings 5.2%',      'finance', 0.024),
    (5,  103, 'Raid Boss: Play Free',         'gaming',  0.140),
    (6,  103, 'Build Your Empire Now',        'gaming',  0.118),
    (7,  104, 'Maldives Getaway Deals',       'travel',  0.072),
    (8,  104, 'Cheap Flights to Tokyo',       'travel',  0.066),
    (9,  105, '30-Minute Meal Kits',          'food',    0.094),
    (10, 105, 'Late-Night Pizza Deal',        'food',    0.101),
    (11, 106, 'Lease an EV Today',            'auto',    0.031),
    (12, 106, 'Trade In, Trade Up',           'auto',    0.028),
    (13, 107, 'Flagship Phone Pre-Order',     'tech',    0.058),
    (14, 107, 'Noise-Cancelling Earbuds',     'tech',    0.063),
    (15, 108, 'Online Therapy, $0 First Week','health',  0.037),
    (16, 108, 'Daily Vitamins Subscription',  'health',  0.041),
    (17, 109, 'Outlet Clearance Blowout',     'retail',  0.070),
    (18, 110, 'Crypto Index Fund',            'finance', 0.022),
    (19, 111, 'Discontinued Console Bundle',  'gaming',  0.090),
    (20, 112, 'Weekend Road Trip Rentals',    'travel',  0.055)
ON CONFLICT (id) DO NOTHING;

COMMIT;
