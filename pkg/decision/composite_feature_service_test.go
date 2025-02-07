/****************************************************************************
 * Copyright 2019-2020, Optimizely, Inc. and contributors                   *
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

package decision

import (
	"errors"
	"testing"

	"github.com/WolffunService/experiment/pkg/decide"
	"github.com/WolffunService/experiment/pkg/decision/reasons"
	"github.com/WolffunService/experiment/pkg/entities"
	"github.com/WolffunService/experiment/pkg/logging"

	"github.com/stretchr/testify/suite"
)

type CompositeFeatureServiceTestSuite struct {
	suite.Suite
	mockFeatureService         *MockFeatureDecisionService
	mockFeatureService2        *MockFeatureDecisionService
	testFeatureDecisionContext FeatureDecisionContext
	options                    *decide.Options
	reasons                    decide.DecisionReasons
}

func (s *CompositeFeatureServiceTestSuite) SetupTest() {
	mockConfig := new(mockProjectConfig)

	s.mockFeatureService = new(MockFeatureDecisionService)
	s.mockFeatureService2 = new(MockFeatureDecisionService)
	s.options = &decide.Options{}
	s.reasons = decide.NewDecisionReasons(s.options)

	// Setup test data
	s.testFeatureDecisionContext = FeatureDecisionContext{
		Feature:       &testFeat3335,
		ProjectConfig: mockConfig,
	}
}

func (s *CompositeFeatureServiceTestSuite) TestGetDecision() {
	// Test that we return the first decision that is made and the next decision service does not get called
	testUserContext := entities.UserContext{
		ID: "test_user_1",
	}

	expectedDecision := FeatureDecision{
		Decision:   Decision{reasons.BucketedIntoVariation},
		Source:     FeatureTest,
		Experiment: testExp1113,
		Variation:  &testExp1113Var2223,
	}
	s.mockFeatureService.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(expectedDecision, s.reasons, nil)

	compositeFeatureService := &CompositeFeatureService{
		featureServices: []FeatureService{
			s.mockFeatureService,
			s.mockFeatureService2,
		},
		logger: logging.GetLogger("sdkKey", "CompositeFeatureService"),
	}
	decision, _, err := compositeFeatureService.GetDecision(s.testFeatureDecisionContext, testUserContext, s.options)
	s.Equal(expectedDecision, decision)
	s.NoError(err)
	s.mockFeatureService.AssertExpectations(s.T())
	s.mockFeatureService2.AssertNotCalled(s.T(), "GetDecision")
}

func (s *CompositeFeatureServiceTestSuite) TestGetDecisionFallthrough() {
	// test that we move onto the next decision service if no decision is made
	testUserContext := entities.UserContext{
		ID: "test_user_1",
	}

	nilDecision := FeatureDecision{}
	s.mockFeatureService.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(nilDecision, s.reasons, nil)

	expectedDecision := FeatureDecision{
		Variation: &testExp1113Var2223,
	}
	s.mockFeatureService2.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(expectedDecision, s.reasons, nil)

	compositeFeatureService := &CompositeFeatureService{
		featureServices: []FeatureService{
			s.mockFeatureService,
			s.mockFeatureService2,
		},
		logger: logging.GetLogger("sdkKey", "CompositeFeatureService"),
	}
	decision, _, err := compositeFeatureService.GetDecision(s.testFeatureDecisionContext, testUserContext, s.options)
	s.Equal(expectedDecision, decision)
	s.NoError(err)
	s.mockFeatureService.AssertExpectations(s.T())
	s.mockFeatureService2.AssertExpectations(s.T())
}

func (s *CompositeFeatureServiceTestSuite) TestGetDecisionReturnsError() {
	// test that we move onto the next decision service if an inner service returns an error
	testUserContext := entities.UserContext{
		ID: "test_user_1",
	}

	shouldBeIgnoredDecision := FeatureDecision{
		Variation: &testExp1113Var2223,
	}
	s.mockFeatureService.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(shouldBeIgnoredDecision, s.reasons, errors.New("Error making decision"))

	expectedDecision := FeatureDecision{
		Variation: &testExp1113Var2224,
	}
	s.mockFeatureService2.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(expectedDecision, s.reasons, nil)

	compositeFeatureService := &CompositeFeatureService{
		featureServices: []FeatureService{
			s.mockFeatureService,
			s.mockFeatureService2,
		},
		logger: logging.GetLogger("sdkKey", "CompositeFeatureService"),
	}
	decision, _, err := compositeFeatureService.GetDecision(s.testFeatureDecisionContext, testUserContext, s.options)
	s.Equal(expectedDecision, decision)
	s.NoError(err)
	s.mockFeatureService.AssertExpectations(s.T())
	s.mockFeatureService2.AssertExpectations(s.T())
}

func (s *CompositeFeatureServiceTestSuite) TestGetDecisionReturnsLastDecisionWithError() {
	// test that GetDecision returns the last decision with error if all decision services return error
	testUserContext := entities.UserContext{
		ID: "test_user_1",
	}

	expectedDecision := FeatureDecision{
		Variation: &testExp1113Var2223,
	}
	s.mockFeatureService.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(expectedDecision, s.reasons, errors.New("Error making decision"))
	s.mockFeatureService2.On("GetDecision", s.testFeatureDecisionContext, testUserContext, s.options).Return(expectedDecision, s.reasons, errors.New("test error"))

	compositeFeatureService := &CompositeFeatureService{
		featureServices: []FeatureService{
			s.mockFeatureService,
			s.mockFeatureService2,
		},
		logger: logging.GetLogger("sdkKey", "CompositeFeatureService"),
	}
	decision, _, err := compositeFeatureService.GetDecision(s.testFeatureDecisionContext, testUserContext, s.options)
	s.Equal(expectedDecision, decision)
	s.Error(err)
	s.Equal(err.Error(), "test error")
	s.mockFeatureService.AssertExpectations(s.T())
	s.mockFeatureService2.AssertExpectations(s.T())
}

func (s *CompositeFeatureServiceTestSuite) TestNewCompositeFeatureService() {
	// Assert that the service is instantiated with the correct child services in the right order
	compositeExperimentService := NewCompositeExperimentService("")
	compositeFeatureService := NewCompositeFeatureService("", compositeExperimentService)
	s.Equal(2, len(compositeFeatureService.featureServices))
	s.IsType(&FeatureExperimentService{compositeExperimentService: compositeExperimentService}, compositeFeatureService.featureServices[0])
	s.IsType(&RolloutService{}, compositeFeatureService.featureServices[1])
}

func TestCompositeFeatureTestSuite(t *testing.T) {
	suite.Run(t, new(CompositeFeatureServiceTestSuite))
}
