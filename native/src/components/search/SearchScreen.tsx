import { useRouter, type Href } from "expo-router";
import SearchIcon from "lucide-react-native/icons/search";
import { useState } from "react";
import { View } from "react-native";

import { groupSearchHits, searchCut, type SearchHit } from "@nexul/client-core/search";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SearchFeed } from "@/components/search/SearchFeed";
import { phoneSearchKinds } from "@/components/search/SearchRow";
import { Input } from "@/components/ui/input";
import { useSearchWorkspace } from "@/hooks/SearchHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";

// Every hit the phone shows has a screen: a pull request opens the ticket it is linked to.
export const searchHitRoute = (hit: SearchHit): Href => {
  if (hit.type === "ticket") return `/board/ticket/${hit.id}`;
  if (hit.type === "pull_request") return `/board/ticket/${hit.ticket_id ?? ""}`;
  if (hit.type === "stack") return `/deploys/stack/${hit.id}`;
  if (hit.type === "channel") return `/chat/${hit.id}`;
  return `/more/docs/${hit.id}`;
};

export const SearchScreen = () => {
  const router = useRouter();
  const workspace = useSelectedWorkspace();
  const [text, setText] = useState("");
  const { data, error, isFetching, settled } = useSearchWorkspace(workspace?.id, text);
  const typed = text.trim().length >= 2;
  const groups = typed && data ? groupSearchHits(data.hits, phoneSearchKinds) : [];
  const waiting = typed && (!settled || isFetching);

  return (
    <View className="flex-1 bg-background">
      <View className="px-4 pt-3 pb-1">
        <Input
          autoFocus
          value={text}
          onChangeText={setText}
          placeholder={workspace ? `Search ${workspace.name}` : "Search"}
          aria-label="Search this workspace"
          returnKeyType="search"
          autoCapitalize="none"
          autoCorrect={false}
          clearButtonMode="while-editing"
        />
      </View>
      {!typed && <EmptyState icon={SearchIcon} title="Search this workspace" message="Tickets, docs, pull requests, stacks and channels you can open." />}
      {waiting && groups.length === 0 && <LoadingDisplay message="Searching" />}
      {typed && error && <ErrorDisplay error={error} />}
      {typed && !waiting && data && groups.length === 0 && (
        <EmptyState icon={SearchIcon} title="No results" message={`Nothing you can open here matches “${text.trim()}”. Try other words.`} />
      )}
      {groups.length > 0 && <SearchFeed groups={groups} cut={searchCut(data, phoneSearchKinds)} onSelect={(hit) => router.push(searchHitRoute(hit), { withAnchor: true })} />}
    </View>
  );
};
