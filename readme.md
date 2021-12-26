# Аренда и бронирование машин

Сервер, на котором можно протестировать запросы: 46.101.82.106:8080

## API

### Авторизация

Создание пользователя

> POST /create_user

Пример запроса:

```json
{
  "username": "vasya",
  "password": "qweasd"
}
```

Пример ответа:

```json
{
  "Username": "vasya"
}
```

Авторизация пользователя

> GET /auth_user

Пример запроса:

```json
{
  "username": "vasya",
  "password": "qweasd"
}
```

Пример ответа:

```json
{
  "token": "token"
}
```

Для всех запросов ниже необходимо использовать заголовок X-Auth с токеном


### Авторизация

Аренда машины

> POST /create_booking

Пример запроса:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999

}
```

Пример ответа:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999,
  "booking_id": 11111
}
```

Проверка машины на доступность

> GET /check_car

Пример запроса:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999
}
```

Пример ответа:

```json
{
  "is_free": true
}
```


Бронирование машины

> POST /create_lease

Пример запроса:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999

}
```

Пример ответа:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999,
  "used_id": 111,
  "lease_id": 11111
}
```

Проверка машины на доступность

> GET /check_lease

Пример запроса:

```json
{
  "car_id": 2222,
  "from_day": 9996,
  "to_day": 9999
}
```

Пример ответа:

```json
{
  "is_free": true
}
```
