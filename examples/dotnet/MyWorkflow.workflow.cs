namespace Dotnet.ActivitySimple;

using Microsoft.Extensions.Logging;
using Temporalio.Workflows;

[Workflow]
public class MyWorkflow
{
  [WorkflowRun]
  public async Task<string> RunAsync(string name)
  {
    // Run a sync static method activity.
    var result = await Workflow.ExecuteActivityAsync(
        () => MyActivities.SayHello(name),
        new()
        {
          StartToCloseTimeout = TimeSpan.FromMinutes(5),
        });
    Workflow.Logger.LogInformation("Activity static method result: {Result}", result);

    // We'll go ahead and return this result
    return result;
  }
}
