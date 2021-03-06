package rmq

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type Delivery interface {
	Payload() string
	Ack() bool
	Reject() bool
	Push() bool
}

type wrapDelivery struct {
	payload     string
	unackedKey  string
	rejectedKey string
	pushKey     string
	delayedKey  string
	redisClient *redis.Client
}

func newDelivery(payload, unackedKey, rejectedKey, pushKey string, delayedKey string, redisClient *redis.Client) *wrapDelivery {
	return &wrapDelivery{
		payload:     payload,
		unackedKey:  unackedKey,
		rejectedKey: rejectedKey,
		pushKey:     pushKey,
		delayedKey:  delayedKey,
		redisClient: redisClient,
	}
}

func (delivery *wrapDelivery) String() string {
	return fmt.Sprintf("[%s %s]", delivery.payload, delivery.unackedKey)
}

func (delivery *wrapDelivery) Payload() string {
	return delivery.payload
}

func (delivery *wrapDelivery) Ack() bool {
	// debug(fmt.Sprintf("delivery ack %s", delivery)) // COMMENTOUT

	result := delivery.redisClient.LRem(context.Background(), delivery.unackedKey, 1, delivery.payload)
	if redisErrIsNil(result) {
		return false
	}

	return result.Val() == 1
}

func (delivery *wrapDelivery) Reject() bool {
	return delivery.move(delivery.rejectedKey)
}

func (delivery *wrapDelivery) Push() bool {
	if delivery.pushKey != "" {
		return delivery.move(delivery.pushKey)
	} else {
		return delivery.move(delivery.rejectedKey)
	}
}

func (delivery *wrapDelivery) move(key string) bool {
	if redisErrIsNil(delivery.redisClient.LPush(context.Background(), key, delivery.payload)) {
		return false
	}

	if redisErrIsNil(delivery.redisClient.LRem(context.Background(), delivery.unackedKey, 1, delivery.payload)) {
		return false
	}

	// debug(fmt.Sprintf("delivery rejected %s", delivery)) // COMMENTOUT
	return true
}
