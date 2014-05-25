redisence
=========

Presence in Redis
------------------

One of the most important and resource intensive operation in chat systems is handling user presence.
Because in order to start the chat one should provide the presence information.
And keeping this data up-to-date is another challenge.

There are 3 main parts for handling user presence,
* Notification
* Persistency
* Data source

#### Starting with number one:
If we calculate the message count with a standart architecture for sending notifications to the online users:
count = (online user count) * (channel participant count) * (events), since every part is a
multiplication even increasing by one will cause huge effects.

#### Number two:
Transient presence data should be commited and updated regularly, after an inactive duration presence data should be set to offline
either by the application or by the persistency layer

#### Number three:
Online clients should notify the presence system with their unique identifier repeatedly. We can call it as heartbeat.



With this package i am gonna try to handle the persistency layer for user presence.




To get the events from the redis database we should uptade the redis config with the following data

# Redis configuration
`redis-cli config set notify-keyspace-events Ex$`

Or
set in redis.conf
`notify-keyspace-events "Ex$"`
