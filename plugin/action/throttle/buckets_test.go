package throttle

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetaActualizeIndex(t *testing.T) {
	type testCase struct {
		curMaxID int
		newMaxID int
		index    int
	}

	tests := []struct {
		name string
		tc   testCase

		wantIndex  int
		wantActual bool
	}{
		{
			name: "same_max_id",
			tc: testCase{
				curMaxID: 10,
				newMaxID: 10,
				index:    5,
			},
			wantIndex:  5,
			wantActual: true,
		},
		{
			name: "not_same_max_id_actual",
			tc: testCase{
				curMaxID: 12,
				newMaxID: 10,
				index:    5,
			},
			wantIndex:  3,
			wantActual: true,
		},
		{
			name: "not_same_max_id_not_actual",
			tc: testCase{
				curMaxID: 30,
				newMaxID: 10,
				index:    5,
			},
			wantIndex:  -15,
			wantActual: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta := bucketsMeta{
				maxID: tt.tc.curMaxID,
			}
			index, actual := meta.actualizeIndex(tt.tc.newMaxID, tt.tc.index)
			require.Equal(t, tt.wantIndex, index, "wrong index")
			require.Equal(t, tt.wantActual, actual, "wrong actuality")
		})
	}
}

func TestRebuildBuckets(t *testing.T) {
	const interval = time.Second

	// equals to current bucketsMeta.timeToBucketID func
	timeToID := func(t time.Time) int {
		return int(t.UnixNano() / interval.Nanoseconds())
	}

	ts, _ := time.Parse(time.RFC3339, "2024-02-12T10:20:30Z")

	type testCase struct {
		b         []int64
		meta      *bucketsMeta
		currentTs time.Time
		ts        time.Time
	}

	tests := []struct {
		name string
		tc   testCase

		wantID      int
		wantBuckets []int64
		wantMinID   int
		wantMaxID   int
	}{
		{
			name: "zero_min_id",
			tc: testCase{
				b: []int64{1, 2, 3},
				meta: &bucketsMeta{
					count:    3,
					interval: interval,
					minID:    0,
				},
				currentTs: ts,
			},
			wantID:      -1,
			wantBuckets: []int64{1, 2, 3},
			wantMinID:   timeToID(ts) - 2, // 2 = count-1
			wantMaxID:   timeToID(ts),
		},
		{
			name: "current_id_not_greater_max_id",
			tc: testCase{
				b: []int64{1, 2, 3},
				meta: &bucketsMeta{
					count:    3,
					interval: interval,
					minID:    timeToID(ts),
				},
				currentTs: ts,
			},
			wantID:      -1,
			wantBuckets: []int64{1, 2, 3},
			wantMinID:   timeToID(ts),
			wantMaxID:   -1,
		},
		{
			name: "current_id_greater_max_id",
			tc: testCase{
				b: []int64{1, 2, 3},
				meta: &bucketsMeta{
					count:    3,
					interval: interval,
					minID:    timeToID(ts),
				},
				currentTs: ts.Add(4 * time.Second),
			},
			wantID:      -1,
			wantBuckets: []int64{3, 0, 0},
			wantMinID:   timeToID(ts.Add(2 * time.Second)),
			wantMaxID:   timeToID(ts.Add(4 * time.Second)),
		},
		{
			name: "ts_id_between_minmax",
			tc: testCase{
				meta: &bucketsMeta{
					interval: interval,
					minID:    timeToID(ts),
					maxID:    timeToID(ts.Add(3 * time.Second)),
				},
				ts: ts.Add(time.Second),
			},
			wantID:    timeToID(ts.Add(time.Second)),
			wantMinID: -1,
			wantMaxID: -1,
		},
		{
			name: "ts_id_not_between_minmax",
			tc: testCase{
				meta: &bucketsMeta{
					interval: interval,
					minID:    timeToID(ts),
					maxID:    timeToID(ts.Add(3 * time.Second)),
				},
				ts: ts.Add(5 * time.Second),
			},
			wantID:    timeToID(ts.Add(3 * time.Second)), // same as maxID
			wantMinID: -1,
			wantMaxID: -1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, b := rebuildBuckets(tt.tc.b, tt.tc.meta, func() int64 { return 0 }, tt.tc.currentTs, tt.tc.ts)

			if tt.wantID != -1 {
				require.Equal(t, tt.wantID, id, "wrong ID")
			}
			if len(tt.wantBuckets) > 0 {
				require.Equal(t, true, slices.Equal(tt.wantBuckets, b), "wrong buckets")
			}
			if tt.wantMinID != -1 {
				require.Equal(t, tt.wantMinID, tt.tc.meta.minID, "wrong buckets min ID")
			}
			if tt.wantMaxID != -1 {
				require.Equal(t, tt.wantMaxID, tt.tc.meta.maxID, "wrong buckets max ID")
			}
		})
	}
}
