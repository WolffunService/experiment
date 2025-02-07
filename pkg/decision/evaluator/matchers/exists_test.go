/****************************************************************************
 * Copyright 2019-2021, Optimizely, Inc. and contributors                   *
 *                                                                          *
 * Licensed under the Apache License, Version 2.0 (the "License");          *
 * you may not use this file except in compliance with the License.         *
 * You may obtain a copy of the License at                                  *
 *                                                                          *
 *    http://www.apache.org/licenses/LICENSE-2.0                            *
 *                                                                          *
 * Unless required by applicable law or agreed to in writing, software      *
 * distributed under the License is distributed on an "AS IS" BASIS,        *
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. *
 * See the License for the specific language governing permissions and      *
 * limitations under the License.                                           *
 ***************************************************************************/

package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/WolffunService/experiment/pkg/entities"
)

var existsMatcher, _ = Get(ExistsMatchType)

func TestExistsMatcher(t *testing.T) {
	condition := entities.Condition{
		Match: "exists",
		Name:  "string_foo",
	}

	// Test match
	user := entities.UserContext{
		Attributes: map[string]interface{}{
			"string_foo": "any_value",
		},
	}

	result, err := existsMatcher(condition, user, nil)
	assert.NoError(t, err)
	assert.True(t, result)

	// Test no match
	user = entities.UserContext{
		Attributes: map[string]interface{}{
			"string_foo1": "not_foo",
		},
	}

	result, err = existsMatcher(condition, user, nil)
	assert.NoError(t, err)
	assert.False(t, result)

	// Test null case
	user = entities.UserContext{
		Attributes: map[string]interface{}{},
	}
	result, err = ExistsMatcher(condition, user, nil)
	assert.NoError(t, err)
	assert.False(t, result)
}
