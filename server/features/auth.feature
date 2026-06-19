Feature: User registration and authentication
  As a person who wants to organise work in ergonomos
  I want to create an account and sign in
  So that my tasks, projects and documents are private to me until I share them

  Background:
    Given a running ergonomos instance

  Scenario: Register a new account
    When I register with email "ada@example.org" and a valid password
    Then my account is created
    And I receive an authentication token
    And my actor handle is "ada" on this instance

  Scenario: Reject duplicate registration
    Given an account already exists for "ada@example.org"
    When I register with email "ada@example.org" and a valid password
    Then registration is rejected as a conflict

  Scenario: Sign in with valid credentials
    Given an account exists for "ada@example.org"
    When I sign in with email "ada@example.org" and the correct password
    Then I receive an authentication token

  Scenario: Reject sign in with a wrong password
    Given an account exists for "ada@example.org"
    When I sign in with email "ada@example.org" and an incorrect password
    Then authentication is rejected
