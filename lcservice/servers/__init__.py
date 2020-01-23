try:
    from .cherrypy import ServeCherryPy
except ImportError:
    ServeCherryPy = None

try:
    from .cloud_function import ServeCloudFunction
except ImportError:
    ServeCloudFunction = None