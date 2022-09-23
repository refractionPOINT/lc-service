"""Reference implementation for LimaCharlie.io services."""

__version__ = "1.9.0"
__author__ = "Maxime Lamothe-Brassard ( Refraction Point, Inc )"
__author_email__ = "maxime@refractionpoint.com"
__license__ = "Apache v2"
__copyright__ = "Copyright (c) 2020 Refraction Point, Inc"

from .service import Service  # noqa: F401
from .service import InteractiveService # noqa: F401
from . import servers         # noqa: F401
