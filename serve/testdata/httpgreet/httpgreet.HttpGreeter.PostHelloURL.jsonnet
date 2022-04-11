// httpgreet.HttpGreeter.PostHelloURL (Unary)

// Input:
// {
//   request: {
//     firstName: "",  // string
//     lastName: "",  // string
//   },
// }

function(input) {
  response: {
    greeting: "Thanks for the post and the path, " + input.request.firstName,  // string
  },
}
