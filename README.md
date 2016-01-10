# smplsms

A simple SMS server for sending text SMS messages using an SMS modem.

The modem code is gratefully stolen from [haxpax/gosms](https://github.com/haxpax/gosms), and therefore this repo retains the same license (GPLv2).
PDU code also gratefully stolen from [ivahaev/gosms](https://github.com/ivahaev/gosms). :)

Post messages as json 
```
$ curl -v -X POST -d @msg.json http://localhost:8951/api/sms
* Hostname was NOT found in DNS cache
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 8951 (#0)
> POST /api/sms HTTP/1.1
> User-Agent: curl/7.35.0
> Host: localhost:8951
> Accept: */*
> Content-Length: 48
> Content-Type: application/x-www-form-urlencoded
> 
* upload completely sent off: 48 out of 48 bytes
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< X-Powered-By: Black Jelly Beans
< Date: Sat, 01 Aug 2015 11:21:16 GMT
< Content-Length: 44
< 
{"message":"message received","status":200}
* Connection #0 to host localhost left intact
$
```
