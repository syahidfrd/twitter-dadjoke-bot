### twitter-dadjoke-bot
- Setup env
```bash
cp .env.example .env
```

- Run app
```bash
go run main.go
```

- Setup webhooks. Ref: [Twitter Account Activity API](https://developer.twitter.com/en/docs/twitter-api/premium/account-activity-api/overview)

- Tips: If you are still running / developing your service on localhost, you can take advantage of services such as [Ngrok](https://ngrok.com) to expose your localhost server to the public Internet. Once you have an Internet-accessible URL, you can add it to the twitter activity webhook.