namespace Dotnet.ActivitySimple;

using Temporalio.Activities;

public class MyActivities
{
  // Activities can be static and/or sync
  [Activity]
  public static string SayHello(string name) => "Hello " + name + "!";
}
