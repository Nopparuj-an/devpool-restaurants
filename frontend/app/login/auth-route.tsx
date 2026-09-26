import { redirect } from "next/navigation";

import { AuthPage } from "@/components/pages/pages";
import { apiGet, getAccount } from "@/lib/api-server";

// Shared by /login and /signup. Only same-site paths are allowed for ?next=.
export async function AuthRoute({
  mode,
  searchParams,
}: {
  mode: "login" | "signup";
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { next: rawNext, error } = await searchParams;
  const next = typeof rawNext === "string" && rawNext.startsWith("/") && !rawNext.startsWith("//") ? rawNext : "/";
  if (await getAccount()) redirect(next);
  const providers = await apiGet<{ google: boolean }>("/auth/providers");
  return (
    <AuthPage
      mode={mode}
      googleEnabled={providers.google}
      error={typeof error === "string" ? error : undefined}
      next={next}
    />
  );
}
