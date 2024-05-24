package lib

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestIsMemoOnce(t *testing.T) {
	// 测试包含 "@once" 的 memo 字符串
	memo := "Meeting @once"
	expected := true
	result := IsMemoOnce(memo)
	if result != expected {
		t.Errorf("IsMemoOnce returned %v for memo '%s', expected %v", result, memo, expected)
	}

	// 测试不包含 "@once" 的 memo 字符串
	memo = "Reminder"
	expected = false
	result = IsMemoOnce(memo)
	if result != expected {
		t.Errorf("IsMemoOnce returned %v for memo '%s', expected %v", result, memo, expected)
	}

	// 测试空字符串
	memo = ""
	expected = false
	result = IsMemoOnce(memo)
	if result != expected {
		t.Errorf("IsMemoOnce returned %v for memo '%s', expected %v", result, memo, expected)
	}

	// 测试包含 "@once" 的 memo 字符串但大小写不一致
	memo = "Task @ONCE"
	expected = false
	result = IsMemoOnce(memo)
	if result != expected {
		t.Errorf("IsMemoOnce returned %v for memo '%s', expected %v", result, memo, expected)
	}
}

func TestShouldExecuteNow(t *testing.T) {
	now := time.Now().UTC()
	loc, _ := time.LoadLocation("Asia/Taipei")
	tt := now.In(loc)
	execTime := tt.Add(5 * time.Hour).Unix()

	testCases := []struct {
		memo     string
		expected bool
	}{
		{
			memo:     fmt.Sprintf("%d@once", execTime), // 未來timestamp
			expected: false,
		},
		{
			memo:     "1630660501@once", // 過期timestamp
			expected: true,
		},
		{
			memo:     "1630660501@task",
			expected: false,
		},
		{
			memo:     "9999999999@task",
			expected: false,
		},
		{
			memo:     "9999999999@once",
			expected: false,
		},
		{
			memo:     "invalid_memo_format",
			expected: false,
		},
		{
			memo:     "9999999999@once@abc",
			expected: false,
		},
		{
			memo:     "$1630660501@once",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.memo, func(t *testing.T) {
			if result := ShouldExecuteNow(tc.memo); result != tc.expected {
				t.Errorf("CheckRightNow(%q) = %v, expected %v", tc.memo, result, tc.expected)
			}

		})
	}
}

func TestShouldExecuteThreeHours(t *testing.T) {
	now := time.Now().UTC()
	loc, _ := time.LoadLocation("Asia/Taipei")
	tt := now.In(loc)
	execTime := tt.Add(5 * time.Hour).Unix()
	execTime1 := tt.Add(-1 * time.Hour).Unix()
	execTime2 := tt.Add(-2 * time.Hour).Unix()
	execTime3 := tt.Add(-3 * time.Hour).Unix()
	execTime4 := tt.Add(-4 * time.Hour).Unix()
	execTime5 := tt.Add(-5 * time.Hour).Unix()

	testCases := []struct {
		memo     string
		expected bool
	}{
		{
			memo:     fmt.Sprintf("%d@once", execTime), // 未來timestamp
			expected: true,
		},
		{
			memo:     fmt.Sprintf("%d@once", execTime1),
			expected: true,
		},
		{
			memo:     fmt.Sprintf("%d@once", execTime2),
			expected: true,
		},
		{
			memo:     fmt.Sprintf("%d@once", execTime3),
			expected: true,
		},
		{
			memo:     fmt.Sprintf("%d@once", execTime4),
			expected: false,
		},
		{
			memo:     fmt.Sprintf("%d@once", execTime5),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.memo, func(t *testing.T) {
			if result := ShouldExecuteThreeHours(tc.memo); result != tc.expected {
				t.Errorf("CheckRightNow(%q) = %v, expected %v", tc.memo, result, tc.expected)
			}

		})
	}
}

func TestSortMapByValue(t *testing.T) {
	maps := map[string]string{
		"pro01": "400003",
		"job01": "400001",
		"ssr01": "400002",
	}

	expected := []string{"400001", "400002", "400003"}

	result := SortMapByValue(maps)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SortMapByValue failed. Expected %v, but got %v", expected, result)
	}
}

func TestMatchJobName(t *testing.T) {
	cases := []struct {
		input    string
		expected struct {
			v1, v2 string
		}
	}{
		{
			input: "M1_DRAW_SCHEDULE",
			expected: struct {
				v1, v2 string
			}{
				v1: "M1",
				v2: "",
			},
		},
		{
			input: "MM1T_DRAW_SCHEDULE",
			expected: struct {
				v1, v2 string
			}{
				v1: "MM1T",
				v2: "",
			},
		},
		{
			input: "BM1E_DRAW_SCHEDULE",
			expected: struct {
				v1, v2 string
			}{
				v1: "BM1E",
				v2: "",
			},
		},
		{
			input: "BM1E_GAME_ROUND_202307181440_ROUND_2_ACTIVE",
			expected: struct {
				v1, v2 string
			}{
				v1: "BM1E",
				v2: "202307181440",
			},
		},
		{
			input: "BM1E_TRIR_GAME_CLOSE_202307181295",
			expected: struct {
				v1, v2 string
			}{
				v1: "BM1E",
				v2: "202307181295",
			},
		},
		{
			input: "BM1S_TRR_GAME_OPEN_202307181294",
			expected: struct {
				v1, v2 string
			}{
				v1: "BM1S",
				v2: "202307181294",
			},
		},
		{
			input: "ABCD_test_123",
			expected: struct {
				v1, v2 string
			}{
				v1: "ABCD",
				v2: "123",
			},
		},
		{
			input: "WXYZ_example",
			expected: struct {
				v1, v2 string
			}{
				v1: "WXYZ",
				v2: "",
			},
		},
		{
			input: "invalid_input",
			expected: struct {
				v1, v2 string
			}{
				v1: "",
				v2: "",
			},
		},
	}

	for _, tc := range cases {
		v1, v2 := MatchJobName(tc.input)
		if v1 != tc.expected.v1 || v2 != tc.expected.v2 {
			t.Errorf("Input: %s, Expected: (%s, %s), Got: (%s, %s)", tc.input, tc.expected.v1, tc.expected.v2, v1, v2)
		}
	}
}

func TestCalculateNextRunTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Taipei")
	now := time.Date(2023, time.July, 26, 12, 0, 0, 0, loc)

	tests := []struct {
		name     string
		next     time.Time
		interval time.Duration
		expected time.Time
	}{
		{
			name:     "Next time before now",
			next:     time.Date(2023, time.July, 26, 11, 0, 0, 0, loc),
			interval: time.Hour,
			expected: time.Date(2023, time.July, 26, 13, 0, 0, 0, loc),
		},
		{
			name:     "Next time after now",
			next:     time.Date(2023, time.July, 26, 14, 0, 0, 0, loc),
			interval: time.Hour,
			expected: time.Date(2023, time.July, 26, 14, 0, 0, 0, loc),
		},
		{
			name:     "With different interval",
			next:     time.Date(2023, time.July, 26, 10, 30, 0, 0, loc),
			interval: 30 * time.Minute,
			expected: time.Date(2023, time.July, 26, 12, 30, 0, 0, loc),
		},
		{
			name:     "With different interval second",
			next:     time.Date(2023, time.July, 26, 12, 0, 0, 0, loc),
			interval: 30 * time.Second,
			expected: time.Date(2023, time.July, 26, 12, 0, 30, 0, loc),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := CalculateNextRunTime(now, test.next, test.interval)
			if !actual.Equal(test.expected) {
				t.Errorf("Expected next execution time: %v, but got: %v", test.expected, actual)
			}
		})
	}
}
