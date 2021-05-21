package request

import (
	"testing"

	"github.com/refractionPOINT/lc-service/lcservice-go/common"
	"github.com/refractionPOINT/lc-service/lcservice-go/service/acker"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestCommandResolverTestSuite(t *testing.T) {
	suite.Run(t, new(RequestResolverTestSuite))
}

type RequestResolverTestSuite struct {
	suite.Suite
	mHandlerGetter mockHandlerGetter
	resolver       requestHandlerResolver
}

func (s *RequestResolverTestSuite) SetupSuite() {
	s.resolver = NewResolver(&s.mHandlerGetter)
}

func (s *RequestResolverTestSuite) SetupTest() {
	s.mHandlerGetter.ExpectedCalls = []*mock.Call{}
}

type mockHandlerGetter struct {
	mock.Mock
}

func (m *mockHandlerGetter) GetHandler(eventType string) (common.ServiceCallback, bool) {
	args := m.Called(eventType)
	return args.Get(0).(common.ServiceCallback), args.Get(1).(bool)
}

func (s *RequestResolverTestSuite) TestGetType() {
	s.Equal("request", s.resolver.GetType())
}

func (s *RequestResolverTestSuite) TestParse() {
	parsedData, err := s.resolver.Parse(common.RequestEvent{})
	s.NoError(err)
	s.Nil(parsedData)

	data := common.Dict{"foo": "bar"}
	parsedData, err = s.resolver.Parse(common.RequestEvent{Data: data})
	s.NoError(err)
	s.Equal(data, parsedData)
}

func (s *RequestResolverTestSuite) TestPreHandleHook() {
	s.Nil(s.resolver.PreHandlerHook(common.Request{}, acker.NoopAcker{}))
}

// TODO test parse