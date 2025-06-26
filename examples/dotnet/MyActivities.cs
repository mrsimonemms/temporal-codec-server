namespace Dotnet.ActivitySimple;

using Temporalio.Activities;

public class MyActivities
{
  private readonly MyDatabaseClient dbClient = new();

  // Activities can be static and/or sync
  [Activity]
  public static string SayHello(string name) => "Hello " + name + "!";

  public class MyDatabaseClient
  {
    public Task<string> SelectValueAsync(string table) =>
        Task.FromResult($"some-db-value from table {table}");
  }
}
