Feature: Data version
  As a job seeker who may have more than one version of Callback on my computer
  I want an older Callback to refuse data that a newer one has updated
  So that an older version never changes or damages data it doesn't understand

  How the command line handles this is in agent-cli.feature (cli-newer-data and cli-older-data).

  Rule: The app checks my data's version before it changes anything

    @status:not-implemented @id:app-newer-data
    Scenario: The app refuses data a newer Callback has updated
      Given my data was last updated by a newer version of Callback than the app
      When I open the app
      Then I'm told my data is from a newer version of Callback, and to update Callback
      And the app doesn't show my board
      And my data is unchanged

    @status:not-implemented @id:app-newer-data-no-backup
    Scenario: The app takes no daily backup of data from a newer Callback
      Given my data was last updated by a newer version of Callback than the app
      And no backup has been taken today
      When I open the app
      Then no daily backup is taken
      And no backup is removed

    @status:not-implemented @id:app-older-data
    Scenario: The app brings older data up to date
      Given my data was last updated by an older version of Callback
      When I open the app
      Then my data is brought up to date
      And the board shows my jobs as usual

    @status:not-implemented @id:app-same-version-data
    Scenario: The app opens data from the same version as usual
      Given my data was last updated by the same version of Callback as the app
      When I open the app
      Then the board shows my jobs as usual
