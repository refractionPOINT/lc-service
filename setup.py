from setuptools import setup

__version__ = "1.8.0"
__author__ = "Maxime Lamothe-Brassard ( Refraction Point, Inc )"
__author_email__ = "maxime@refractionpoint.com"
__license__ = "Apache v2"
__copyright__ = "Copyright (c) 2020 Refraction Point, Inc"

setup( name = 'lcservice',
       version = __version__,
       description = 'Reference implementation for LimaCharlie.io services.',
       url = 'https://limacharlie.io',
       author = __author__,
       author_email = __author_email__,
       license = __license__,
       packages = [ 'lcservice', 'lcservice.servers', 'lcservice' ],
       zip_safe = True,
       install_requires = [ 'limacharlie', 'cherrypy', 'pyyaml' ],
       long_description = 'Reference implementation for LimaCharlie.io services, allowing anyone to extend and automate services around LimaCharlie.'
)