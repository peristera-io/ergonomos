Feature: Task management
  As a signed-in user
  I want to create, retrieve, and list my own tasks
  So that I can track my work without seeing anyone else's

  Background:
    Given a running ergonomos instance
    And I have registered as "ada@example.org"

  Scenario: Create a task
    When I create a task with title "Write acceptance tests"
    Then the task is created
    And the task has an opaque, globally-unique identifier
    And the task owner is my actor
    And the task is not marked done

  Scenario: Reject a task with an empty title
    When I create a task with title ""
    Then the request is rejected as invalid

  Scenario: Retrieve my task by ID
    Given I have created a task with title "Write acceptance tests"
    When I request that task by its ID
    Then the task is returned
    And the task title is "Write acceptance tests"

  Scenario: Retrieving an unknown task is not found
    When I request a task with an unknown ID
    Then the task is not found

  Scenario: List my tasks
    Given I have created a task with title "Write acceptance tests"
    And I have created a task with title "Review PR"
    When I request my tasks
    Then I receive a list containing both tasks

  Scenario: Another user cannot retrieve my task
    Given I have created a task with title "Write acceptance tests"
    And another user "grace@example.org" has signed in
    When that user requests my task by its ID
    Then the task is not found

  Scenario: My task list contains only my tasks
    Given I have created a task with title "Write acceptance tests"
    And another user "grace@example.org" has signed in
    And that user has created a task with title "Grace's secret"
    When I request my tasks
    Then I receive a list containing only my task "Write acceptance tests"

  Scenario: Update a task's title
    Given I have created a task with title "Draft"
    When I change that task's title to "Final draft"
    Then the task is returned
    And the task title is "Final draft"

  Scenario: Reject updating a task to an empty title
    Given I have created a task with title "Draft"
    When I change that task's title to ""
    Then the request is rejected as invalid

  Scenario: Another user cannot update my task
    Given I have created a task with title "Draft"
    And another user "grace@example.org" has signed in
    When that user changes my task's title to "Hijacked"
    Then the task is not found

  Scenario: Complete a task
    Given I have created a task with title "Write acceptance tests"
    When I complete that task
    Then the task is returned
    And the task is marked done

  Scenario: Another user cannot complete my task
    Given I have created a task with title "Write acceptance tests"
    And another user "grace@example.org" has signed in
    When that user completes my task
    Then the task is not found

  Scenario: Reject unauthenticated task update
    Given I have created a task with title "Draft"
    When I change that task's title to "Final" without a token
    Then the request is unauthorized

  Scenario: Delete a task
    Given I have created a task with title "Write acceptance tests"
    When I delete that task
    Then the task is deleted
    When I request that task by its ID
    Then the task is not found

  Scenario: Another user cannot delete my task
    Given I have created a task with title "Write acceptance tests"
    And another user "grace@example.org" has signed in
    When that user deletes my task
    Then the task is not found

  Scenario: Deleting an unknown task is not found
    When I delete a task with an unknown ID
    Then the task is not found

  Scenario: Reject unauthenticated task deletion
    Given I have created a task with title "Write acceptance tests"
    When I delete that task without a token
    Then the request is unauthorized

  Scenario: Reject unauthenticated task creation
    When I create a task without a token
    Then the request is unauthorized

  Scenario: Reject unauthenticated task retrieval
    Given I have created a task with title "Write acceptance tests"
    When I request that task without a token
    Then the request is unauthorized
