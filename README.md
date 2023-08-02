# redis-lock
Fast distributed lock using Redis

This lock is implemented for single instances of Redis, using a non-polling system for the fastest unlock notification.  When contending for the lock, first a pubsub subscription is established, then the lock contention itself occurs.  If the lock is acquired successfully, the subscription is closed and the lock is held.  If the lock is not acquired, the contender waits on the subscription for a notification of release of the lock by the existing lock holder.  At this time, all lock contenders will re-bid to acquire the lock.  When the lock is held, it will be kept alive periodically, to ensure that in untimely death of the process, the lock is eventually automatically released.
