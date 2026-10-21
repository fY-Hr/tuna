create table if not exists users (
    id bigint generated always as identity primary key,
    username varchar(50) unique not null,
    email varchar(255) unique not null,
    password_hash text not null,
    bio text,
    avatar_url text,
    role varchar(20) default 'user' not null,
    created_at timestamptz default now() not null,
    updated_at timestamptz default now() not null
);

create table if not exists wave_streams(
    id bigint generated always as identity primary key,
    name varchar(100) unique not null,
    description text,
    creator_id bigint references users(id) on delete set null,
    requires_approval boolean default false not null,
    created_at timestamptz default now() not null,
    updated_at timestamptz default now() not null
);

create table if not exists waves(
    id bigint generated always as identity primary key,
    user_id bigint not null references users(id) on delete cascade,
    content text not null,
    is_archive boolean default false not null,
    stream_id bigint references wave_streams(id) on delete cascade,
    created_at timestamptz default now() not null,
    updated_at timestamptz default now() not null
);