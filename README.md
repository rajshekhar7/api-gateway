# api-gateway

Run main.go
`$ go run main.go`

Get the access token from `/oauth` with username(email) and password. We also have to provide the client_id and client_secret as stored in .env file.

`seedUser.json` contains some initial user data.


```
$ curl "http://localhost:8000/oauth?grant_type=password&client_id=APP01&client_secret=APPSEC&username=steven@gmail.com&password=password1" | jq

{
  "access_token": "3VQEXRDCMOI0BTO9GR1AZA",
  "expires_in": 300,
  "token_type": "Bearer"
}
```

Use this access_token to access the user information field from `/home` 

```
$ curl "http://localhost:8000/home?access_token=STSYUQWHNQ-BHLNZOS5IEW" | jq

{
  "id": "0001",
  "username": "Steven victor",
  "email": "steven@gmail.com",
  "password": "$2a$10$2ebVzQPyvFUB3mgDd9d/HuiT3fRbuTUTn6swJwQfY.ydnwSh0DjxC"
}

```


