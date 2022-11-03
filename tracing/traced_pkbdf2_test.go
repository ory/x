// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ory/x/tracing"
)

func TestComparePKDBF2(t *testing.T) {
	iter := 10
	keylenght := 10
	hasher := tracing.NewTracedPKDBF2([]byte("1234567890"), func(ctx context.Context) int {
		return iter
	}, func(ctx context.Context) int {
		return keylenght
	})

	expectedPassword := "hello world"
	expectedPasswordHash, err := hasher.Hash(context.TODO(), []byte(expectedPassword))
	assert.NoError(t, err)
	assert.NotNil(t, expectedPasswordHash)

	expectedTagsSuccess := map[string]interface{}{
		tracing.PKDBFIterationsTagName: int(iter),
		tracing.PKDBFKeyLengthTagName:  int(keylenght),
	}

	expectedTagsError := map[string]interface{}{
		tracing.PKDBFIterationsTagName: int(iter),
		tracing.PKDBFKeyLengthTagName:  int(keylenght),
	}

	testCases := []struct {
		testDescription  string
		providedPassword string
		expectedTags     map[string]interface{}
		shouldError      bool
	}{
		{
			testDescription:  "should not return an error if hash of provided password matches hash of expected password",
			providedPassword: expectedPassword,
			expectedTags:     expectedTagsSuccess,
			shouldError:      false,
		},
		{
			testDescription:  "should return an error if hash of provided password does not match hash of expected password",
			providedPassword: "some invalid password",
			expectedTags:     expectedTagsError,
			shouldError:      true,
		},
	}

	for _, test := range testCases {
		t.Run(test.testDescription, func(t *testing.T) {
			hash, err := hasher.Hash(context.TODO(), []byte(test.providedPassword))
			assert.NoError(t, err)
			assert.NotNil(t, hash)

			mockedTracer.Reset()

			err = hasher.Compare(context.TODO(), expectedPasswordHash, []byte(test.providedPassword))
			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			spans := mockedTracer.FinishedSpans()
			assert.Len(t, spans, 2)
			span1 := spans[0]
			span2 := spans[1]

			assert.Equal(t, tracing.PKDBF2HashOpName, span1.OperationName)
			assert.Equal(t, test.expectedTags, span1.Tags())

			assert.Equal(t, tracing.PKDBF2CompareOpName, span2.OperationName)
			if test.shouldError {
				assert.Equal(t, map[string]interface{}{
					"error": true,
				}, span2.Tags())
			} else {
				assert.Empty(t, span2.Tags())
			}
		})
	}
}
