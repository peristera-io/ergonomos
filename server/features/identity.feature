Feature: Authenticated identity
  As a signed-in person
  I want the server to recognise me from my token
  So that my tasks, projects and documents stay tied to my account

  Background:
    Given a running ergonomos instance

  Scenario: Look up my own profile
    Given I have registered as "ada@example.org"
    When I request my profile with my token
    Then my profile is returned
    And my actor handle is "ada" on this instance

  Scenario: Reject a request with no token
    When I request my profile without a token
    Then the request is unauthorized

  Scenario: Reject a request with an unknown token
    When I request my profile with the token "not-a-real-token"
    Then the request is unauthorized
