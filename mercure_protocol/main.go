package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Subscriber struct {
	IPAddress string
	Topics    map[string]struct{} // Set-like structure for topics
	Channel   chan string
}

type Hub struct {
	sync.RWMutex
	subscribers map[string]*Subscriber            // Subscribers by IP address
	topics      map[string]map[string]*Subscriber // Subscribers grouped by topic
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]*Subscriber),
		topics:      make(map[string]map[string]*Subscriber),
	}
}

func (h *Hub) Broadcast(topic string, data string) error {
	h.RLock()
	defer h.RUnlock()

	subscribers, exists := h.topics[topic]
	if !exists {
		return nil
	}

	for _, subscriber := range subscribers {
		fmt.Println("Broadcasting to", subscriber.IPAddress, topic, data)
		select {
		case subscriber.Channel <- data:
		default:
		}
	}

	return nil
}

func (h *Hub) AddSubscriber(ipAddress string, topics []string) (chan string, error) {
	h.Lock()
	defer h.Unlock()

	if _, exists := h.subscribers[ipAddress]; exists {
		return nil, errors.New("subscriber already exists")
	}

	channel := make(chan string, 100)
	subscriber := &Subscriber{
		IPAddress: ipAddress,
		Topics:    make(map[string]struct{}),
		Channel:   channel,
	}

	h.subscribers[ipAddress] = subscriber

	for _, topic := range topics {
		subscriber.Topics[topic] = struct{}{}
		if _, exists := h.topics[topic]; !exists {
			h.topics[topic] = make(map[string]*Subscriber)
		}
		h.topics[topic][ipAddress] = subscriber
	}

	return channel, nil
}

func (h *Hub) RemoveSubscriber(subscriber *Subscriber) error {
	h.Lock()
	defer h.Unlock()

	if _, exists := h.subscribers[subscriber.IPAddress]; !exists {
		return errors.New("subscriber does not exist")
	}

	for topic := range subscriber.Topics {
		delete(h.topics[topic], subscriber.IPAddress)
		if len(h.topics[topic]) == 0 {
			delete(h.topics, topic)
		}
	}

	close(subscriber.Channel)
	delete(h.subscribers, subscriber.IPAddress)

	return nil
}

func (h *Hub) GetSubscriptions(subscriber *Subscriber) error {
	h.RLock()
	defer h.RUnlock()

	if _, exists := h.subscribers[subscriber.IPAddress]; !exists {
		return errors.New("subscriber does not exist")
	}

	// Subscriber already has their subscriptions in `subscriber.Topics`
	return nil
}

func (h *Hub) CloseAndDisconnectSubscribers() error {
	h.Lock()
	defer h.Unlock()

	for _, subscriber := range h.subscribers {
		close(subscriber.Channel)
	}

	h.subscribers = make(map[string]*Subscriber)
	h.topics = make(map[string]map[string]*Subscriber)

	return nil
}

func main() {
	// Example usage
	hub := NewHub()
	user1, err := hub.AddSubscriber("127.0.0.1", []string{"topic1", "topic2"})
	if err != nil {
		panic(err)
	}
	user2, err := hub.AddSubscriber("127.0.0.2", []string{"topic1"})
	if err != nil {
		panic(err)
	}

	// wait for messages in all users
	go func() {
		for {
			select {
			case msg := <-user1:
				fmt.Println("user1:", msg)
			case msg := <-user2:
				fmt.Println("user2:", msg)
			}
		}
	}()

	go hub.Broadcast("topic1", "hello")
	go hub.Broadcast("topic2", "world")

	// Wait for messages
	time.Sleep(1 * time.Second)
}
