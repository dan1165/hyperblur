"""openblur specific endpoints"""

import sanic
from .misc import misc_bp

openblur = sanic.Blueprint.group(misc_bp, url_prefix="/openblur")
