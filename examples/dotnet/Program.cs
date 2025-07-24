using Microsoft.Extensions.Logging;
using Temporalio.Client;
using Temporalio.Worker;
using Temporalio.Converters;
using Dotnet.ActivitySimple;
using TemporalCodec.Algorithms.AES;

// Create a client to localhost on default namespace
var client = await TemporalClient.ConnectAsync(new("localhost:7233")
{
  DataConverter = DataConverter.Default with
  {
    PayloadCodec = await AESCodec.Create(Environment.GetEnvironmentVariable("KEYS_PATH"))
  },
  LoggerFactory = LoggerFactory.Create(builder =>
      builder.
          SetMinimumLevel(LogLevel.Information)),
});

async Task RunWorkerAsync()
{
  // Cancellation token cancelled on ctrl+c
  using var tokenSource = new CancellationTokenSource();
  Console.CancelKeyPress += (_, eventArgs) =>
  {
    tokenSource.Cancel();
    eventArgs.Cancel = true;
  };

  // Create an activity instance with some state
  var activities = new MyActivities();

  // Run worker until cancelled
  Console.WriteLine("Running worker");
  using var worker = new TemporalWorker(
      client,
      new TemporalWorkerOptions(taskQueue: "activity-simple-sample").
          AddActivity(MyActivities.SayHello).
          AddWorkflow<MyWorkflow>());
  try
  {
    await worker.ExecuteAsync(tokenSource.Token);
  }
  catch (OperationCanceledException)
  {
    Console.WriteLine("Worker cancelled");
  }
}

async Task ExecuteWorkflowAsync()
{
  Console.WriteLine("Executing workflow");

  Random rnd = new();
  int id = rnd.Next(1000, 9999);

  string res = await client.ExecuteWorkflowAsync(
      (MyWorkflow wf) => wf.RunAsync("World"),
      new(id: $"activity-simple-workflow-{id}", taskQueue: "activity-simple-sample"));

  Console.WriteLine($"Result: {res}");
}

switch (args.ElementAtOrDefault(0))
{
  case "worker":
    await RunWorkerAsync();
    break;
  case "workflow":
    await ExecuteWorkflowAsync();
    break;
  default:
    throw new ArgumentException("Must pass 'worker' or 'workflow' as the single argument");
}
