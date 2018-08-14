#!/bin/bash

riff service create correlator --image dev.local/correlator:0.0.1

riff channel create replies --cluster-bus stub

riff service subscribe correlator --input replies

