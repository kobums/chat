# Chat 서버 (go-socket.io 기반)

이 프로젝트는 Go와 Socket.IO를 이용하여 만든 간단한 채팅 서버입니다.

## 설치 및 실행

1. go run main

2. localhost:9000 서버 접속

3. 각 다른 브라우저나 크롬 시크릿모드로 여러 창을 띄움

4. 로그인 후 채팅

## 인증 / 권한

- 로그인: 처음 로그인한 `loginid`는 그 비밀번호(bcrypt 해시)로 등록되고, 이후 같은 `loginid`는 같은 비밀번호로만 로그인할 수 있습니다. 계정은 메모리에만 저장되어 서버 재시작 시 초기화됩니다. 실패 시 응답 `status`가 `0`입니다.
- 비밀번호 방: `makeroom`의 `password`가 있으면 bcrypt 해시로 저장되고, `joinroom` 요청의 `password`가 맞아야 입장할 수 있습니다(실패 시 `joinroom_fail` 이벤트). `listroom` 응답에는 비밀번호가 포함되지 않습니다.
- `ban` / `kick`: `{"id": "<대상 소켓 id>", "room": <방 번호>}` — 방장(방을 만든 `loginid`)만 가능합니다. 대상은 대기실로 이동하고, ban된 `loginid`는 그 방에 다시 입장할 수 없습니다.
