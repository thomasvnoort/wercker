package main

import (
  "fmt"
  "log"
  "strings"
  "code.google.com/p/go.net/websocket"
  "code.google.com/p/go-uuid/uuid"
)


// Session is our class for interacting with a running Docker container.
type Session struct {
  wsUrl string
  ws *websocket.Conn
  ch chan string
}


// CreateSession based on a docker api endpoint and container ID.
func CreateSession (endpoint string, containerID string) (*Session) {
  wsEndpoint := strings.Replace(endpoint, "tcp://", "ws://", 1)
  wsQuery := "stdin=1&stderr=1&stdout=1&stream=1"
  wsUrl := fmt.Sprintf("%s/containers/%s/attach/ws?%s",
                       wsEndpoint, containerID, wsQuery)

  ch := make(chan string)

  return &Session{wsUrl: wsUrl, ws:nil, ch:ch}
}


// ReadToChan reads on a websocket forever, writing to a channel
func ReadToChan(ws *websocket.Conn, ch chan string) {
  var data string
  for {
    err := websocket.Message.Receive(ws, &data)
    if err != nil {
      log.Fatalln(err)
    }
    ch <- data
  }
}


// Attach begins reading on the websocket and writing to the internal channel.
func (s *Session) Attach() (*Session, error) {
  ws, err := websocket.Dial(s.wsUrl, "", "http://localhost/")
  if err != nil {
    return s, err
  }
  s.ws = ws

  go ReadToChan(s.ws, s.ch)
  return s, nil
}


// Send an array of commands.
func (s *Session) Send(commands ...string) {
  for i := range commands {
    fmt.Println("send: ", commands[i])
    err := websocket.Message.Send(s.ws, commands[i] + "\n")
    if err != nil {
      log.Fatalln(err)
    }
  }
}

// SendChecked sends commands, waits for them to complete and returns the
// exit status and output
func (s *Session) SendChecked(commands []string) (int, []string, error) {
  var exitCode int
  rand := uuid.NewRandom().String()
  check := false
  recv := []string{}

  s.Send(commands)
  s.Send([]string{fmt.Sprintf("echo %s $?", rand)})

  // This is relatively naive and will break if the messages returned aren't
  // complete lines, if this becomes a problem we'll have to buffer it.
  for check != true {
    line := <- s.ch
    fmt.Print("recv: ", line)
    if strings.HasPrefix(line, rand) {
      check = true
      _, err := fmt.Sscanf(line, "%s %d\n", &rand, &exitCode)
      if err != nil {
        return exitCode, recv, err
      }
    } else {
      recv = append(recv, line)
    }
  }
  return exitCode, recv, nil
}
