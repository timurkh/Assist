package main

import (
	"context"
	"log"
	"strconv"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/patrickmn/go-cache"
)

type Notification struct {
	Time  time.Time `json:"time"`
	Title string    `json:"title"`
	Text  string    `json:"text"`
}

type Notifications struct {
	notificationsCache *cache.Cache
	dev                bool
	messaging          *messaging.Client
	userTokens         *cache.Cache
}

func InitNotifications(fireapp *firebase.App, dev bool) (*Notifications, error) {

	ctx := context.Background()

	nc := cache.New(24*time.Hour, 10*time.Minute)
	ut := cache.New(24*time.Hour, 10*time.Minute)

	messaging, err := fireapp.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	ntfs := Notifications{notificationsCache: nc, dev: dev, messaging: messaging, userTokens: ut}

	return &ntfs, nil

}

func (ntfs *Notifications) logDev(format string, v ...interface{}) {
	if ntfs.dev {
		if ntfs.dev {
			log.Printf(format, v...)
		}
	}
}

func (ntfs *Notifications) createNotification(userIds []string, title string, notification string) {

	if ntfs.dev {
		log.Printf("createNotification %v:%v %+v\n", title, notification, userIds)
	}

	n := Notification{
		time.Now(),
		title,
		notification,
	}

	for _, userId := range userIds {
		var notifications []Notification
		ns, found := ntfs.notificationsCache.Get(userId)
		if !found {
			notifications = make([]Notification, 0)
		} else {
			notifications = ns.([]Notification)
		}
		notifications = append(notifications, n)
		ntfs.notificationsCache.Set(userId, notifications, cache.DefaultExpiration)

		ntfs.sendMessage(userId, n, len(notifications))
	}
}

func (ntfs *Notifications) sendMessage(userId string, n Notification, count int) {

	token := ntfs.GetUserToken(userId)

	if token != "" {
		ctx := context.Background()

		message := &messaging.Message{
			Data: map[string]string{
				"count": strconv.Itoa(count),
				"title": n.Title,
				"text":  n.Text,
				"time":  n.Time.Format("2006.01.02 15:04:05"),
			},
			Token: token,
		}

		response, err := ntfs.messaging.Send(ctx, message)
		if err != nil {
			log.Println(err)
		}

		if ntfs.dev {
			log.Printf("sendMessage(userId=%v, token=%v, title=%v) %v: %+v\n", userId, token, n.Title, response)
		}
	}
}

func (ntfs *Notifications) GetNotificationsCount(userId string) int {
	ns, found := ntfs.notificationsCache.Get(userId)
	if ntfs.dev {
		log.Printf("GetNotificationsCount %v %v %+v\n", userId, found, ns)
	}
	if !found {
		return 0
	} else {
		return len(ns.([]Notification))
	}
}

func (ntfs *Notifications) GetNotifications(userId string) []Notification {
	ns, found := ntfs.notificationsCache.Get(userId)
	if ntfs.dev {
		log.Printf("GetNotifications %v %v %+v\n", userId, found, ns)
	}
	if found {
		return ns.([]Notification)
	}
	return nil
}

func (ntfs *Notifications) MarkNotificationsDelivered(userId string) {
	ntfs.notificationsCache.Delete(userId)
}

func (ntfs *Notifications) GetUserToken(userId string) string {

	v, found := ntfs.userTokens.Get(userId)
	if found {
		return v.(string)
	}
	return ""
}

func (ntfs *Notifications) SetUserToken(userId string, token string) {
	if ntfs.dev {
		log.Printf("SetUserToken(%v, %v)\n", userId, token)
	}
	ntfs.userTokens.Set(userId, token, cache.DefaultExpiration)
}

func (ntfs *Notifications) DeleteUserToken(userId string) {
	if ntfs.dev {
		log.Printf("DeleteUserToken(%v)\n", userId)
	}
	ntfs.userTokens.Delete(userId)
}
