package main

import (
	"fmt"
	"net/http"
    "time"
    "math/rand"
    "strconv"
	"encoding/json"
	"github.com/gorilla/websocket"	
	uuid "github.com/satori/go.uuid"
	
)

type ClientManager struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

type Client struct {
    id     string
    socket *websocket.Conn
    send   chan []byte
}

type Message struct {
    Sender    string `json:"sender,omitempty"`
    Recipient string `json:"recipient,omitempty"`
    Content string `json:"content,omitempty"`

}
type KoinStructure struct {
    prevBitcoin float64
    curBitcoin float64
    prevLitecoin float64
    curLitecoin float64
}

var KoinData = KoinStructure {
    prevBitcoin: 0,
    curBitcoin:  0,
    prevLitecoin:  0,
    curLitecoin:  0,
}

var manager = ClientManager{
    broadcast:  make(chan []byte),
    register:   make(chan *Client),
    unregister: make(chan *Client),
    clients:    make(map[*Client]bool),
}

func (manager *ClientManager) start() {
    for {
        select {
        case conn := <-manager.register:
            manager.clients[conn] = true
            updateClient(conn)
            //bitcoinMessage, _ := json.Marshal(
            //    &Message{Content: "bitcoin~getbitcoin~Bitcoin is $10,000"})
           
            //manager.sendToOne(bitcoinMessage, conn)

            //litecoinMessage, _ := json.Marshal(
            //    &Message{Content: "litecoin~getlitecoin~1 Bitcoin is $10,000."})
            
            //manager.sendToOne(litecoinMessage, conn)

        case conn := <-manager.unregister:
            if _, ok := manager.clients[conn]; ok {
                close(conn.send)
                delete(manager.clients, conn)
                //jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
                //manager.sendToAll(jsonMessage, conn)
            }
        case message := <-manager.broadcast:
            for conn := range manager.clients {
                select {
                case conn.send <- message:
                default:
                    close(conn.send)
                    delete(manager.clients, conn)
                }
            }
        default:
            updateClient(nil)
            time.Sleep(5000 * time.Millisecond)              
        }
    }
}

func (manager *ClientManager) sendToAll(message []byte) {
    for conn := range manager.clients {
        conn.send <- message
    }
}

func (manager *ClientManager) sendToOne(message []byte, client *Client) {
    client.send <- message
}

func (c *Client) read() {
    defer func() {
        manager.unregister <- c
        c.socket.Close()
    }()

    for {
        _, message, err := c.socket.ReadMessage()
        if err != nil {
            manager.unregister <- c
            c.socket.Close()
            break
        }

        bitcoinMessage, _ := json.Marshal(
           &Message{Content: "bitcoin~getbitcoin~"+string(message)}) 
        manager.broadcast <- bitcoinMessage
    }
}

func (c *Client) write() {
    defer func() {
        c.socket.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.socket.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            c.socket.WriteMessage(websocket.TextMessage, message)
        }
    }
}

func main() {
    fmt.Println("Starting application...")
    go manager.start()
    http.HandleFunc("/ws", wsPage)
    http.ListenAndServe(":12345", nil)
}
func updateClient(client *Client){

    KoinData.prevBitcoin = KoinData.curBitcoin
    KoinData.curBitcoin = (rand.Float64() * 3000) + 7000

    KoinData.prevLitecoin = KoinData.curLitecoin
    KoinData.curLitecoin = (rand.Float64() * 3000) + 7000

    if(KoinData.prevBitcoin != KoinData.curBitcoin){
         bitcoinMessage, _ := json.Marshal(
            &Message{Content: "bitcoin~getbitcoin~" + strconv.FormatFloat(KoinData.curBitcoin, 'f', 4, 64) + "~"+ time.Now().Format("2006-01-02 15:04:05")})
        
        if(client == nil){
            manager.sendToAll(bitcoinMessage)
        }else {
            manager.sendToOne(bitcoinMessage,client)
        }
    }
    if(KoinData.prevLitecoin != KoinData.curLitecoin){
        litecoinMessage, _ := json.Marshal(
            &Message{Content: "litecoin~getlitecoin~"+strconv.FormatFloat(KoinData.curLitecoin, 'f', 4, 64) +"~"+ time.Now().Format("2006-01-02 15:04:05")})
        
        
        if(client == nil){
            manager.sendToAll(litecoinMessage)
        }else {
            manager.sendToOne(litecoinMessage,client)
        }
     }
}
func wsPage(res http.ResponseWriter, req *http.Request) {
    conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
    if error != nil {
        http.NotFound(res, req)
        return
    }
    client := &Client{id: uuid.Must(uuid.NewV4()).String(), socket: conn, send: make(chan []byte)}

    manager.register <- client

    go client.read()
    go client.write()
}