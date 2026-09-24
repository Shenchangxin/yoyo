// @ts-nocheck
import { useQuery } from "@tanstack/react-query";

import { accountFileStorageUsageQueryKey } from "@yingce/lib/account-storage-usage";
import { getAccountFileStorageUsage } from "@yingce/services/api/resources";

export function useAccountFileStorageUsage(enabled = true) {
    return useQuery({
        queryKey: accountFileStorageUsageQueryKey,
        queryFn: getAccountFileStorageUsage,
        enabled,
        staleTime: 30_000,
        refetchOnMount: "always",
    });
}
