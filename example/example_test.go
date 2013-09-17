package example

import (
	"testing"

	"code.google.com/p/gomock/gomock"

	mockfmt "fmt" // mock

	zmq "github.com/alecthomas/gozmq" // mock
	"labix.org/v2/mgo"                // mock
)

func TestSend10(t *testing.T) {
	// Create a gomock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup the mock zmq to use the controller
	zmq.MOCK().SetController(ctrl)

	// Setup the mock fmt to use the controller
	mockfmt.MOCK().SetController(ctrl)

	// Create a mock *zmq.Context and *zmq.Socket
	ctxt := &zmq.Context{}
	skt := &zmq.Socket{}

	// We expect the package function NewContext to be called
	zmq.EXPECT().NewContext().Return(ctxt, nil)

	// We expect the NewSocket method of our Context to be called
	ctxt.EXPECT().NewSocket(zmq.REQ).Return(skt, nil)

	// We expect the Socket to be connected to two endpoints
	skt.EXPECT().Connect("tcp://127.0.0.1:1234").Return(nil)
	skt.EXPECT().Connect("tcp://127.0.0.1:5678").Return(nil)

	// fmt.Sprintf should be called to create the message
	mockfmt.EXPECT().Sprintf("msg %d", gomock.Any()).Return("fakeMsg").Times(10)

	// And we expect the Socket to be used to send/recv 10 times.
	skt.EXPECT().Send(gomock.Any(), zmq.SendRecvOption(0)).Return(nil).Times(10)
	skt.EXPECT().Recv(zmq.SendRecvOption(0)).Return([]byte{}, nil).Times(10)

	// Now the expectations are registered in the controller, actually call the
	// function under test ...
	err := send10()

	// We expect to get a nil error back
	if err != nil {
		t.Errorf("send10 return an error: %s", err)
	}
}

func TestMongo(t *testing.T) {
	// Create a gomock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup the mock mgo to use the controller
	mgo.MOCK().SetController(ctrl)

	// Setup the mock fmt to use the controller
	mockfmt.MOCK().SetController(ctrl)

	// Create a mock *mgo.Session, *mgo.Database, *mgo.Collection, and
	// *mgo.Query
	session := &mgo.Session{}
	database := &mgo.Database{}
	collection := &mgo.Collection{}
	query := &mgo.Query{}

	// We expect Dial against localhost
	mgo.EXPECT().Dial("localhost").Return(session, nil)

	// We expect the session to be closed
	session.EXPECT().Close()

	// We expect the named database to be opened
	session.EXPECT().DB("database").Return(database)

	// We also expect the named collection to be opened
	database.EXPECT().C("collection").Return(collection)

	// We then expect a query to be created against the collection, using nil to
	// ask for all entries
	collection.EXPECT().Find(nil).Return(query)

	// Finally we expect the query to be asked for all the matches
	query.EXPECT().All(gomock.Any()).Return(nil)

	// We expect fmt.Printf to be called to print a debug message
	mockfmt.EXPECT().Printf("found %d results", gomock.Any())

	// Now the expectations are registered in the controller, actually call the
	// function under test ...
	_, err := mongo("database", "collection")

	// We don't expect an error
	if err != nil {
		t.Errorf("mongo returned an error: %s", err)
	}
}
