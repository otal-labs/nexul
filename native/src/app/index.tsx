import { Redirect } from "expo-router";

// The web home is a landing hero; the phone opens on Inbox.
export default function Index() {
  return <Redirect href="/inbox" />;
}
