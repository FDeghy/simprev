package main

import "time"

const (
	MAX_IDLE_TIME  = 120 * time.Second
	MAX_IDLE_CONNS = 64
	TIMEOUT        = 15 * time.Second
)
