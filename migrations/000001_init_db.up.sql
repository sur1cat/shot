CREATE TABLE users (
                       id BIGSERIAL PRIMARY KEY,
                       username TEXT UNIQUE NOT NULL,
                       email TEXT UNIQUE NOT NULL,
                       password TEXT NOT NULL,
                       created_at TIMESTAMP NOT NULL,
                       updated_at TIMESTAMP NOT NULL
);

CREATE TABLE links (
                       id BIGSERIAL PRIMARY KEY,
                       user_id BIGINT NOT NULL REFERENCES users(id),
                       original_url TEXT NOT NULL,
                       short_code TEXT UNIQUE NOT NULL,
                       created_at TIMESTAMP NOT NULL,
                       expires_at TIMESTAMP,
                       click_count INT NOT NULL DEFAULT 0
);

CREATE TABLE click_stats (
                             id BIGSERIAL PRIMARY KEY,
                             link_id BIGINT NOT NULL REFERENCES links(id),
                             clicked_at TIMESTAMP NOT NULL,
                             referrer_url TEXT,
                             user_agent TEXT,
                             ip_address TEXT
);

CREATE TABLE tags (
                      id BIGSERIAL PRIMARY KEY,
                      name TEXT UNIQUE NOT NULL,
                      created_at TIMESTAMP NOT NULL
);

CREATE TABLE link_tags (
                           id BIGSERIAL PRIMARY KEY,
                           link_id BIGINT NOT NULL REFERENCES links(id),
                           tag_id BIGINT NOT NULL REFERENCES tags(id)
);

-- Индексы
CREATE INDEX idx_links_user_id ON links(user_id);
CREATE INDEX idx_click_stats_link_id ON click_stats(link_id);
CREATE INDEX idx_link_tags_link_id ON link_tags(link_id);
CREATE INDEX idx_link_tags_tag_id ON link_tags(tag_id);