package example

import (
	"fmt"

	zmq "github.com/alecthomas/gozmq"
	"labix.org/v2/mgo"
)

func send10() error {
	context, err := zmq.NewContext()
	if err != nil {
		return err
	}

	socket, err := context.NewSocket(zmq.REQ)
	if err != nil {
		return err
	}

	if err := socket.Connect("tcp://127.0.0.1:1234"); err != nil {
		return err
	}

	if err := socket.Connect("tcp://127.0.0.1:5678"); err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("msg %d", i)

		err := socket.Send([]byte(msg), 0)
		if err != nil {
			return err
		}

		_, err = socket.Recv(0)
		if err != nil {
			return err
		}
	}

	return nil
}

type Data struct {
	Value int
}

func mongo(db, coll string) ([]Data, error) {
	var result []Data

	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, fmt.Errorf("Dial error: %s\n", err)
	}
	defer session.Close()

	database := session.DB(db)
	collection := database.C(coll)

	err = collection.Find(nil).All(&result)
	if err != nil {
		return nil, err
	}

	fmt.Printf("found %d results", len(result))

	return result, nil
}
