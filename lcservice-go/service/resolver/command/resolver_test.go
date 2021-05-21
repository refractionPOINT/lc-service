package command

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	lc "github.com/refractionPOINT/go-limacharlie/limacharlie"
	"github.com/refractionPOINT/lc-service/lcservice-go/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestCommandResolverTestSuite(t *testing.T) {
	suite.Run(t, new(CommandResolverTestSuite))
}

type CommandResolverTestSuite struct {
	suite.Suite
	mCommandDescritorsGetter mockCommandsDescritorGetter
	resolver                 commandHandlerResolver
}

type emptyLogger struct{}

func (l emptyLogger) Log(string) {}

type mockCommandsDescritorGetter struct {
	mock.Mock
}

func (m *mockCommandsDescritorGetter) GetCommandDescriptors() []common.CommandDescriptor {
	args := m.Called()
	return args.Get(0).([]common.CommandDescriptor)
}

type mockRequestAcker struct {
	mock.Mock
}

func (m *mockRequestAcker) Ack(req common.Request) error {
	args := m.Called(req)
	return args.Error(0)
}

func (s *CommandResolverTestSuite) SetupSuite() {
	s.resolver = NewResolver(&s.mCommandDescritorsGetter, &emptyLogger{})
}

func (s *CommandResolverTestSuite) SetupTest() {
	s.mCommandDescritorsGetter.ExpectedCalls = []*mock.Call{}
}

func (s *CommandResolverTestSuite) TestGetType() {
	s.Equal("command", s.resolver.GetType())
}

func (s *CommandResolverTestSuite) TestParse() {
	parsedData, err := s.resolver.Parse(common.RequestEvent{})
	s.NoError(err)
	s.Nil(parsedData)

	data := common.Dict{"foo": "bar"}
	parsedData, err = s.resolver.Parse(common.RequestEvent{Data: data})
	s.NoError(err)
	s.Equal(data, parsedData)
}

func (s *CommandResolverTestSuite) TestGetCommandNameMissing() {
	s.Nil(s.resolver.Get(common.RequestEvent{}))
}

func (s *CommandResolverTestSuite) TestGetNoCommands() {
	s.mCommandDescritorsGetter.On("GetCommandDescriptors").Return([]common.CommandDescriptor{})
	s.Nil(s.resolver.Get(common.RequestEvent{
		Data: common.Dict{
			"command_name": "testCommand",
		},
	}))
	s.mCommandDescritorsGetter.AssertExpectations(s.T())
}

func (s *CommandResolverTestSuite) TestGetCommandNotFound() {
	s.mCommandDescritorsGetter.On("GetCommandDescriptors").Return([]common.CommandDescriptor{
		{
			Name: "test_command",
		},
	})
	s.Nil(s.resolver.Get(common.RequestEvent{
		Data: common.Dict{
			"command_name": "testCommand",
		},
	}))
	s.mCommandDescritorsGetter.AssertExpectations(s.T())
}

func dummyHandler(common.Request) common.Response {
	return common.Response{}
}

func (s *CommandResolverTestSuite) TestGetCommandFound() {
	s.mCommandDescritorsGetter.On("GetCommandDescriptors").Return([]common.CommandDescriptor{
		{
			Name:    "test_command",
			Handler: dummyHandler,
		},
	})
	handler := s.resolver.Get(common.RequestEvent{
		Data: common.Dict{
			"command_name": "test_command",
		},
	})
	s.Equal(runtime.FuncForPC(reflect.ValueOf(dummyHandler).Pointer()), runtime.FuncForPC(reflect.ValueOf(handler).Pointer()))
	s.mCommandDescritorsGetter.AssertExpectations(s.T())
}

func (s *CommandResolverTestSuite) TestPreHandlerHookNoOrg() {
	mReqAcker := mockRequestAcker{}
	s.NoError(s.resolver.PreHandlerHook(common.Request{}, &mReqAcker))
}

func (s *CommandResolverTestSuite) TestPreHandlerHookWithOrgError() {
	req := common.Request{
		Org: &lc.Organization{},
	}
	mReqAcker := mockRequestAcker{}
	mReqAcker.On("Ack", req).Return(fmt.Errorf("oops"))

	err := s.resolver.PreHandlerHook(req, &mReqAcker)
	s.EqualError(err, "oops")

	mReqAcker.AssertExpectations(s.T())
}

func (s *CommandResolverTestSuite) TestPreHandlerHookWithOrgNoError() {
	req := common.Request{
		Org: &lc.Organization{},
	}
	mReqAcker := mockRequestAcker{}
	mReqAcker.On("Ack", req).Return(nil)

	s.NoError(s.resolver.PreHandlerHook(req, &mReqAcker))

	mReqAcker.AssertExpectations(s.T())
}
