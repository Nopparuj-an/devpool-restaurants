"use client";

import { PasswordForm, ProfileForm } from "@/components/app/account-forms";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import type { Account, ActionResult } from "@/lib/types";

export function AccountScreen({
  account,
  onSaveName,
  onSetPassword,
}: {
  account: Account;
  onSaveName: (name: string) => ActionResult;
  onSetPassword: (next: string) => ActionResult;
}) {
  return (
    <>
      <SiteHeader account={account} />
      <Page>
        <PageTitle title="Account" />
        <div className="max-w-3xl">
          <ProfileForm account={account} onSave={onSaveName} />
          <PasswordForm hasPassword={account.has_password} onSave={onSetPassword} />
        </div>
      </Page>
    </>
  );
}
