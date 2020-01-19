import lcservice

TEST_SECRET = 'test-secret'

def test_create_server():
    svc = lcservice.Service( 'test-service', None )
    lcservice.servers.ServeCherryPy( svc, interface = '0.0.0.0', port = 8888 )

if __name__ == '__main__':
    test_create_server()