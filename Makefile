.PHONY: run run-log

run:
	@{ \
	  go run . stun 2>&1 | sed 's/^/[STUN] /' & \
	  sleep 2; \
	  go run . peer 2>&1 | sed 's/^/[PEER-A] /' & \
	  sleep 2; \
	  go run . peer 2>&1 | sed 's/^/[PEER-B] /' & \
	  wait; \
	}

run-log:
	@mkdir -p logs
	@{ \
	  go run . stun 2>&1 \
	    | sed 's/^/[STUN] /' \
	    | tee logs/stun.log & \
	  sleep 2; \
	  go run . peer 2>&1 \
	    | sed 's/^/[PEER-A] /' \
	    | tee logs/peer-a.log & \
	  sleep 2; \
	  go run . peer 2>&1 \
	    | sed 's/^/[PEER-B] /' \
	    | tee logs/peer-b.log & \
	  wait; \
	}
