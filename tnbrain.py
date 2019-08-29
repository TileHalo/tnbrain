#!/usr/bin/env python3
from urllib import request

class TacnetPasser():
    url = "http://scout.polygame.fi/api/msg?msg=%s"

    """TacnetPasser sends packages received through serial to the havu"""

    def __init__(self):
        """TODO: to be defined. """
        .__init__(self)

    def pass_message(self, msg)
        """Passes message to havu"""
        urllib.request.urlopen(self.url % msg)

    def parse_message(self, msg)
        """Parses message from raw bytestream"""
            
