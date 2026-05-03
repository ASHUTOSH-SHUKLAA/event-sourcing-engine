# Music Streaming Backend

A music streaming backend built with Go, Gin, PostgreSQL, and Redis for event queuing.

## Architecture

### Repository Pattern
The backend follows a clean repository pattern with separate repositories for different domains:

- `UserRepository` - User management
- `SubscriptionRepository` - Subscription management
- `EventRepository` - Event sourcing
- `MusicRepository` - Song and analytics data

### Event Queue
- **Redis-backed Queue**: Persistent event processing using Redis
- **Asynchronous Processing**: Events are queued and processed in the background
- **Database Updates**: Event handlers update PostgreSQL projections

### Services
- `EventQueueService` - Asynchronous event processing with Redis persistence
- `TokenService` - JWT authentication

## Database tables

Run the SQL files in `migrations/` against the PostgreSQL database pointed to by `DATABASE_URL`.

- `users`: login accounts and profile names.
- `subscriptions`: current subscription projection for each user.
- `subscription_events`: event log for subscription and player events.
- `liked_songs`: liked tracks per user.
- `playlists`: user-created playlists.
- `player_state`: latest player state per user.
- `song_play_counts`: realtime play-count analytics by song id.
- `provider_songs`: songs uploaded by service providers.

## Deployment notes

For local development, `DATABASE_URL` can point to a local PostgreSQL database. For deployment, create a managed PostgreSQL database on a provider such as Supabase, Neon, Railway, Render PostgreSQL, or AWS RDS, run the migrations there, and set the deployed backend `DATABASE_URL` to that cloud database connection string.

Set `CORS_ALLOWED_ORIGINS` to your frontend URL when deployed, for example:

```env
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
```
