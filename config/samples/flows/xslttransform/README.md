The XSLTTransform object operates in two modes:
1. If the `sink` paremeter is set in the spec, the XSLTTransform object will
   transform the incoming event and send the result to the `sink`
2. If the `sink` paremeter is NOT set in the spec, the XSLTTransform object
   will transform the incoming event and send the result as a reply to the 
   event sender/source.

This directory contains examples of both modes.
