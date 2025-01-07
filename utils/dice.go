package utils

import (
	"context"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

type Dice struct {
	Number int
	Sides  int
}

func ParseDice(_ context.Context, me *xrpc.AuthInfo, feedPost *bsky.FeedPost) []Dice {
	s := feedPost.Text
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@"+me.Handle)
	s = strings.TrimSpace(s)

	var diceMatcher = regexp.MustCompile(`(\d*)d(\d+)`)
	diceStrs := diceMatcher.FindAllString(s, -1)

	var dicePool []Dice
	for _, diceStr := range diceStrs {
		captureGroups := diceMatcher.FindStringSubmatch(diceStr)
		if len(captureGroups) == 2 {
			sides, _ := strconv.Atoi(captureGroups[1])

			dicePool = append(dicePool, Dice{Number: 1, Sides: sides})
		} else if len(captureGroups) == 3 {
			number, _ := strconv.Atoi(captureGroups[1])
			sides, _ := strconv.Atoi(captureGroups[2])

			dicePool = append(dicePool, Dice{Number: number, Sides: sides})

		}
	}

	return dicePool
}

func RollDice(dice Dice) []int {
	var results []int
	for i := 0; i < dice.Number; i++ {
		results = append(results, rollDie(dice.Sides))
	}
	return results
}

func rollDie(sides int) int {
	return rand.Intn(sides) + 1
}

func Sum(nums []int) int {
	result := 0
	for _, num := range nums {
		result += num
	}
	return result
}
