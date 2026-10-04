"use client";

import { AuthForm, type AuthInput } from "@/components/app/auth-form";
import { Page, SiteHeader } from "@/components/app/site-header";
import type { ActionResult } from "@/lib/types";

export function AuthScreen({
  mode,
  googleEnabled,
  error,
  next,
  onSubmit,
}: {
  mode: "login" | "signup";
  googleEnabled: boolean;
  error?: string;
  next?: string;
  onSubmit: (input: AuthInput) => ActionResult;
}) {
  return (
    <>
      <SiteHeader account={null} />
      <Page narrow>
        <div className="pt-4 sm:pt-10">
          <AuthForm mode={mode} googleEnabled={googleEnabled} error={error} next={next} onSubmit={onSubmit} />
        </div>
      </Page>
    </>
  );
}
