from typing import NamedTuple, Optional


class Signpost(NamedTuple):
    title: str
    description: Optional[str] = None
