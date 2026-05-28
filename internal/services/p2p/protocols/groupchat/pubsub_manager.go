package groupchat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const TopicPrefix = "groupchat-"

type MessageHandler func(msg *GroupMessage)

type PubSubManager struct {
	host              host.Host
	pubsub            *pubsub.PubSub
	topics            map[string]*pubsub.Topic
	subs              map[string]*pubsub.Subscription
	handlers          map[string]MessageHandler
	mu                sync.RWMutex
	ctx               context.Context
	localPeerID       string
	JoinGroupCallback func(groupUUID string, msg *GroupMessage)
}

type GroupMessage struct {
	MessageUUID  string `json:"message_uuid"`
	GroupUUID    string `json:"group_uuid"`
	FromPeerID   string `json:"from_peer_id"`
	Content      string `json:"content"`
	ContentType  string `json:"content_type"`
	Metadata     string `json:"metadata"`
	Timestamp    int64  `json:"timestamp"`
	LamportClock uint64 `json:"lamport_clock"`
	Signature    []byte `json:"signature"`
}

func NewPubSubManager(ctx context.Context, h host.Host, ps *pubsub.PubSub, localPeerID string) *PubSubManager {
	return &PubSubManager{
		ctx:         ctx,
		host:        h,
		pubsub:      ps,
		topics:      make(map[string]*pubsub.Topic),
		subs:        make(map[string]*pubsub.Subscription),
		handlers:    make(map[string]MessageHandler),
		localPeerID: localPeerID,
	}
}

func (m *PubSubManager) JoinGroup(groupUUID string, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	topicName := TopicPrefix + groupUUID

	if _, exists := m.topics[topicName]; exists {
		return nil
	}

	topic, err := m.pubsub.Join(topicName)
	if err != nil {
		return fmt.Errorf("ошибка подключения к topic %s: %w", topicName, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		return fmt.Errorf("ошибка подписки на topic %s: %w", topicName, err)
	}

	m.topics[topicName] = topic
	m.subs[topicName] = sub
	m.handlers[topicName] = handler

	go m.listenMessages(topicName, sub, handler, groupUUID)

	log.Printf("[GroupChat/PubSub] ✅ Подключён к группе %s (topic: %s)", groupUUID[:8], topicName)
	return nil
}

func (m *PubSubManager) LeaveGroup(groupUUID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	topicName := TopicPrefix + groupUUID

	if sub, exists := m.subs[topicName]; exists {
		sub.Cancel()
		delete(m.subs, topicName)
	}

	if topic, exists := m.topics[topicName]; exists {
		_ = topic.Close()
		delete(m.topics, topicName)
	}

	delete(m.handlers, topicName)
	log.Printf("[GroupChat/PubSub] 🚪 Покинул группу %s", groupUUID[:8])
	return nil
}

func (m *PubSubManager) PublishMessage(groupUUID string, msg *GroupMessage) error {
	m.mu.RLock()
	topicName := TopicPrefix + groupUUID
	topic, exists := m.topics[topicName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("не подключён к группе %s", groupUUID)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	if err := topic.Publish(m.ctx, data); err != nil {
		return fmt.Errorf("ошибка публикации: %w", err)
	}

	return nil
}

func (m *PubSubManager) listenMessages(topicName string, sub *pubsub.Subscription, handler MessageHandler, groupUUID string) {
	for {
		msg, err := sub.Next(m.ctx)
		if err != nil {
			if m.ctx.Err() != nil {
				return
			}
			log.Printf("[GroupChat/PubSub] ⚠️ Ошибка получения сообщения из %s: %v", topicName, err)
			continue
		}

		if msg.ReceivedFrom == m.host.ID() {
			continue
		}

		var groupMsg *GroupMessage
		if err := json.Unmarshal(msg.Data, &groupMsg); err != nil {
			log.Printf("[GroupChat/PubSub] ⚠️ Ошибка десериализации: %v", err)
			continue
		}

		if handler != nil {
			handler(groupMsg)
		}

		if m.JoinGroupCallback != nil {
			m.JoinGroupCallback(groupUUID, groupMsg)
		}
	}
}
