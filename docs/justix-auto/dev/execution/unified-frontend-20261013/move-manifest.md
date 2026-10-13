# Exact P3 source move manifest

Base: 743bd6c4accadcd50afb59534bb66dabe2f3baf8. Every row is owned by P3.
Use apply_patch Move to for moves with a narrow context hunk, preserving bytes
unless the packet explicitly names a semantic adaptation. No shell mv/cp, broad
source rewrite, or tests that merely assert this manifest. The four replace-entry
rows are deleted, not copied: transfer their composition to portal sections.tsx.
All directories are ordinary modules, never separate package/workspace entries.

| Action | Source | Destination | Source SHA-256 |
|---|---|---|---|
| move | `web/apps/realization/src/accountant.test.tsx` | `web/apps/portal/src/features/realization/accountant.test.tsx` | `4d5154dfa8246341e6635ab0460f543303a8e66b1a333d93cc0bddc4b84be6cc` |
| move | `web/apps/realization/src/add-company.test.tsx` | `web/apps/portal/src/features/realization/add-company.test.tsx` | `f68631ec7463829e356311eba0772ab80f76116c4013e1d0e8457b65fe78092a` |
| move | `web/apps/realization/src/cash-sale.test.tsx` | `web/apps/portal/src/features/realization/cash-sale.test.tsx` | `3ebc43f1e8880dbfc750dfee6faebe59313c6f1a5b815d6be41f0146d9f731d9` |
| move | `web/apps/realization/src/commercial-access.test.tsx` | `web/apps/portal/src/features/realization/commercial-access.test.tsx` | `97efcc4c1779fc68165d56af156ca7b86f36af9e1630a898fb99ce62523e1f2f` |
| move | `web/apps/realization/src/consignment-workspace.test.tsx` | `web/apps/portal/src/features/realization/consignment-workspace.test.tsx` | `09ffb8f52f6f7112f1f68186bc74fc116cf9629c5f52da7a14701eeeaa003900` |
| move | `web/apps/realization/src/consignment-workspace.tsx` | `web/apps/portal/src/features/realization/consignment-workspace.tsx` | `79f0fff93a5d764f006c5292ca7f779acf88bdffad6e4589e33b9650b6f95413` |
| move | `web/apps/realization/src/contract-workspace.test.tsx` | `web/apps/portal/src/features/realization/contract-workspace.test.tsx` | `c419e2053b5880ea3ca351c5407f7fdddd1f5b8fe897c9348a54b5d3d7be0574` |
| move | `web/apps/realization/src/contract-workspace.tsx` | `web/apps/portal/src/features/realization/contract-workspace.tsx` | `9a0ec867b5457f8e8cd90e96f5dd199ac6e087622acbe839115ca915373ceca5` |
| move | `web/apps/realization/src/customer-passport.test.tsx` | `web/apps/portal/src/features/realization/customer-passport.test.tsx` | `c73edcd47d915c51d086050f18ac021311aa2d6d3397c9c8bafd04f72c70e1ae` |
| move | `web/apps/realization/src/customer-passport.tsx` | `web/apps/portal/src/features/realization/customer-passport.tsx` | `5ae4b91972fe77efcb6451f4228ff0760acceec8d5625e8d789e3f3444e8ed68` |
| move | `web/apps/realization/src/data.ts` | `web/apps/portal/src/features/realization/data.ts` | `d7e5e3c58c26c8d17ddeb9de411267a0a73186036a7dfe7fa7aa04a11bfb0e39` |
| move | `web/apps/realization/src/design.css` | `web/apps/portal/src/features/realization/design.css` | `b1ca9a01f8481b4ace9c28be8fbd83c5d82d297377dc2f3670cb7d93f272ceda` |
| move | `web/apps/realization/src/installment-calculator.test.ts` | `web/apps/portal/src/features/realization/installment-calculator.test.ts` | `6825f75a29e5f35245994c2d55721204cef3428395fcd99f1ec2e20c69618f97` |
| move | `web/apps/realization/src/installment-calculator.ts` | `web/apps/portal/src/features/realization/installment-calculator.ts` | `17c9d2c1df8ba3566d78c3a1c26efcc84f7ef03ff5e1af059a5c818865ea5e74` |
| move | `web/apps/realization/src/installment-payment-form.test.tsx` | `web/apps/portal/src/features/realization/installment-payment-form.test.tsx` | `64c819a939656bcab08672c84cce6f2b58f72afedaabe9ffbd102692b655872a` |
| move | `web/apps/realization/src/installment-payment-form.tsx` | `web/apps/portal/src/features/realization/installment-payment-form.tsx` | `3fd28216b41bc5baa47cbb25d5c4a6fb45f8a71e4f42a882335c500e0a3b02a5` |
| move | `web/apps/realization/src/installment-payment-review.test.tsx` | `web/apps/portal/src/features/realization/installment-payment-review.test.tsx` | `a7c1bea2718cd8b14bf80d3a3c3f88240a3b217ba92ef6b7fee3db31d27cb6a1` |
| move | `web/apps/realization/src/installment-payment-review.tsx` | `web/apps/portal/src/features/realization/installment-payment-review.tsx` | `f6a7bf07f956d3bba7d4e1bac0de7fdf2e187506b6d00fbaac0047fb1c7cb43e` |
| move | `web/apps/realization/src/installment-percent.test.ts` | `web/apps/portal/src/features/realization/installment-percent.test.ts` | `e735f1e5003fc9fc87364cb703cd528f34cb4141dea159aa29c6a5737fe5c1f7` |
| move | `web/apps/realization/src/installment-percent.ts` | `web/apps/portal/src/features/realization/installment-percent.ts` | `a496f204fb3d39b4116114c1fdae5d246eb3ef02baf6d6586fdabb1c6fd3f509` |
| move | `web/apps/realization/src/installment-servicing.test.tsx` | `web/apps/portal/src/features/realization/installment-servicing.test.tsx` | `ad26eb9c2f94c32a27a52ec6184096e4b35472918a24a2f2cbaa282044b5a581` |
| move | `web/apps/realization/src/installment-servicing.tsx` | `web/apps/portal/src/features/realization/installment-servicing.tsx` | `4b7001d7152c02b654b244886a966327eb3bed5aa002cd9d308e4d57e67d9afc` |
| move | `web/apps/realization/src/installment-terms-form.tsx` | `web/apps/portal/src/features/realization/installment-terms-form.tsx` | `7fd46c445e61d2a3800e7347f5f8d009b48cacb487307a14dff682eb85051ad7` |
| replace-entry | `web/apps/realization/src/main.tsx` | `web/apps/portal/src/sections.tsx` (composition only) | `bc144eab1a949aeaefb75ab7db7fbad3e0259c51a8cbc900ee766d4184912f90` |
| move | `web/apps/realization/src/money.test.tsx` | `web/apps/portal/src/features/realization/money.test.tsx` | `ea4f6ffff0d9e295f3d7d72056d63648bfb2e8817986817a3ae4e4dd1b896e69` |
| move | `web/apps/realization/src/navigation.test.tsx` | `web/apps/portal/src/features/realization/navigation.test.tsx` | `e9db114d225130cd5a11a06f916e1a5d6661189aec58a1db96032455caf23017` |
| move | `web/apps/realization/src/navigation.tsx` | `web/apps/portal/src/features/realization/navigation.tsx` | `512b94b94cfad408b983e379565695259e2c4de2a2d2fb8e9c0cc35170b548db` |
| move | `web/apps/realization/src/order-actions.tsx` | `web/apps/portal/src/features/realization/order-actions.tsx` | `61532b42ab99347c4f126df52d6599b04735dfa742a125a3fc0cbdf8c5eb3698` |
| move | `web/apps/realization/src/order-allocation.test.tsx` | `web/apps/portal/src/features/realization/order-allocation.test.tsx` | `34d817eddc03d44aad5acda6b758957bbd0cba176c61d98dfd42c5691a5dda19` |
| move | `web/apps/realization/src/order-create.test.tsx` | `web/apps/portal/src/features/realization/order-create.test.tsx` | `3ca61eec350031a407c2518a8292dd9275336b28a8ae06b989f5350eda87a5bf` |
| move | `web/apps/realization/src/order-create.tsx` | `web/apps/portal/src/features/realization/order-create.tsx` | `cf77da93968659671f9454274991a8b306e6ef1b8125b927abea7ae6490f753d` |
| move | `web/apps/realization/src/order-fulfilment.test.tsx` | `web/apps/portal/src/features/realization/order-fulfilment.test.tsx` | `d571e48e03d645b227fd4410d8325e7c90cd9b55259b7b2e4a3de0baccb7bcee` |
| move | `web/apps/realization/src/order-fulfilment.tsx` | `web/apps/portal/src/features/realization/order-fulfilment.tsx` | `94c384cb633aba94f38d6035bfc2bdaa22a1dc1f729e554c7e112b4ae8996b63` |
| move | `web/apps/realization/src/order-shipping.test.tsx` | `web/apps/portal/src/features/realization/order-shipping.test.tsx` | `e7c081893d0a7fc244ace324248280df8e113d28435b1429b734191da3a018de` |
| move | `web/apps/realization/src/order-workspace.test.tsx` | `web/apps/portal/src/features/realization/order-workspace.test.tsx` | `10c74bff2ee25de18c3ad95729567709a457e8255875802abff607aad0b67ced` |
| move | `web/apps/realization/src/order-workspace.tsx` | `web/apps/portal/src/features/realization/order-workspace.tsx` | `aa32dfaa08e74c9557a81d0e95bd1df87c4ce2cec721fd74cd1688c44a80bc44` |
| move | `web/apps/realization/src/pages/billing.tsx` | `web/apps/portal/src/features/realization/pages/billing.tsx` | `562a80d03badae9f2a69d5255956c2e7bdf9e82c7d067f43da74e01290700d1f` |
| move | `web/apps/realization/src/pages/crm.tsx` | `web/apps/portal/src/features/realization/pages/crm.tsx` | `8cd20d7d4478ca232e962b0f63e0e6299bc8601914a9f05159dcdf3ecdc9108a` |
| move | `web/apps/realization/src/pages/inventory.tsx` | `web/apps/portal/src/features/realization/pages/inventory.tsx` | `167c0ad5d452b2a16d6cd6137d5ac81adeba0921c2f456ced4030aebb9051471` |
| move | `web/apps/realization/src/pages/offers.tsx` | `web/apps/portal/src/features/realization/pages/offers.tsx` | `e3f6eb06a5160172b842d80359542723681962c730ed041d11c069d98afefeb4` |
| move | `web/apps/realization/src/pages/partners-finance.tsx` | `web/apps/portal/src/features/realization/pages/partners-finance.tsx` | `5c1b1aeb8fadb6e1609bd69394d61656864bee6482dbe7dd953c6cf26aa2c157` |
| move | `web/apps/realization/src/pages/partners.tsx` | `web/apps/portal/src/features/realization/pages/partners.tsx` | `66f7a8c5a8d29f5dbc184c14706fda51cc45125b74a28404b5817d45b9f1ad10` |
| move | `web/apps/realization/src/pages/purchases.tsx` | `web/apps/portal/src/features/realization/pages/purchases.tsx` | `7af3ff64da42f928f298c408d0f3f5c9cf8004a22ddb8cfa8c28cd26b1779217` |
| move | `web/apps/realization/src/pages/retail.tsx` | `web/apps/portal/src/features/realization/pages/retail.tsx` | `91275c7514212e483b608067b9fcc4baf7c16b4b7127ab03a8473d0c64b861ac` |
| move | `web/apps/realization/src/pages/sales.tsx` | `web/apps/portal/src/features/realization/pages/sales.tsx` | `e647568877079d2e65219da054875e8759771d3e87a876704d583b73b1897eb2` |
| move | `web/apps/realization/src/pages/settings.tsx` | `web/apps/portal/src/features/realization/pages/settings.tsx` | `5025d68fc5c7b6182025d83a9e21f95939078d4de3d42ff6d8348c98712bb94d` |
| move | `web/apps/realization/src/pages/stock.tsx` | `web/apps/portal/src/features/realization/pages/stock.tsx` | `157e3e0aed7c91f23ad6bdf608495fa1612bf2c2dcceeab19a4d93f83fb595c6` |
| move | `web/apps/realization/src/pages/trade.tsx` | `web/apps/portal/src/features/realization/pages/trade.tsx` | `d129488996057f3b6f7b65d0358ac470bfc75bced927b004daf205a9b7163ba7` |
| move | `web/apps/realization/src/purchases.test.tsx` | `web/apps/portal/src/features/realization/purchases.test.tsx` | `53164c701366ba231511887997c74f97842bc187036c4ac9a9d4268d0e0f4bc4` |
| move | `web/apps/realization/src/receipt-colors.test.tsx` | `web/apps/portal/src/features/realization/receipt-colors.test.tsx` | `006fb12eefc1d2099469c2f26e4bd8eda0a8bc206ab351f3db6f0523b1745553` |
| move | `web/apps/realization/src/receipt-identification.test.tsx` | `web/apps/portal/src/features/realization/receipt-identification.test.tsx` | `41f7c8243c7e336632d9df1a3e8b4b133d967c0d23f4e146956088a61d879c5d` |
| move | `web/apps/realization/src/receipt-identification.tsx` | `web/apps/portal/src/features/realization/receipt-identification.tsx` | `0ae7dba2ed8faab425ad852c932b328f8225893a25c9c9e8a6c5b9e6a8ca6bf4` |
| move | `web/apps/realization/src/reservation-payment.test.tsx` | `web/apps/portal/src/features/realization/reservation-payment.test.tsx` | `f673c1fbbaaf335f4c1ca6b7411c6e67cbeae5ff13a6fdf63a50bf2e434fe0cd` |
| move | `web/apps/realization/src/reservation-payment.tsx` | `web/apps/portal/src/features/realization/reservation-payment.tsx` | `a46c819bef4872b758d93b8270ba2bc26428484995c73afe028c6a27178a10a4` |
| move | `web/apps/realization/src/reservation-workspace.test.tsx` | `web/apps/portal/src/features/realization/reservation-workspace.test.tsx` | `449ba81d950bde35ad8d8f67e2be0f3e9e19def3a7e5aacc85b33ea953bd08ef` |
| move | `web/apps/realization/src/reservation-workspace.tsx` | `web/apps/portal/src/features/realization/reservation-workspace.tsx` | `a8904acffb3476fbb111f9a44f260c36d0211f6711bbd1824563fce68d68819c` |
| move | `web/apps/realization/src/retail-actions.tsx` | `web/apps/portal/src/features/realization/retail-actions.tsx` | `ff385d4d33baf40c86b6a3d7c71c0712c2b553ec6694ba3aa6ef613d2f07e604` |
| move | `web/apps/realization/src/sale-colors.test.tsx` | `web/apps/portal/src/features/realization/sale-colors.test.tsx` | `3353de3b383c87a865c46d1df0e143dd93cede2de7b116e9bd952613e9954fbd` |
| move | `web/apps/realization/src/sale-form.test.tsx` | `web/apps/portal/src/features/realization/sale-form.test.tsx` | `67640f5c434268fb339fc9ed9071c927a56f8017d33ea269271e637a71386757` |
| move | `web/apps/realization/src/sale-form.tsx` | `web/apps/portal/src/features/realization/sale-form.tsx` | `bddff200a6cd3e5841889976499c26193715b3380ce81d0501a1316eaa988657` |
| move | `web/apps/realization/src/sale-insurance.test.tsx` | `web/apps/portal/src/features/realization/sale-insurance.test.tsx` | `23dd27ca3ab268e2c6fba06325e0aebc2aee80027e41faa175a89f6afd4b06c2` |
| move | `web/apps/realization/src/sale-insurance.tsx` | `web/apps/portal/src/features/realization/sale-insurance.tsx` | `91c303a7d0d92d9ce16f2250ba2c2ca06b443d58aee7411e27d99e055172ab07` |
| move | `web/apps/realization/src/sale-offer-entry.test.tsx` | `web/apps/portal/src/features/realization/sale-offer-entry.test.tsx` | `fb1f85131557542f67fdcf6044ac37ddf5406427ecfb810a911c0f2450e47616` |
| move | `web/apps/realization/src/sale-offer-form.test.tsx` | `web/apps/portal/src/features/realization/sale-offer-form.test.tsx` | `2b0160d125c4f5c16f6fb888d8d84d5e9c155b74cdcdabf7ef8fe9491bd13b4b` |
| move | `web/apps/realization/src/sale-offer-picker.tsx` | `web/apps/portal/src/features/realization/sale-offer-picker.tsx` | `03bd09f8d11bc9dc83c7bb83904812291db51219e77ce7d03381429e929b496b` |
| move | `web/apps/realization/src/sale-offer-source.test.ts` | `web/apps/portal/src/features/realization/sale-offer-source.test.ts` | `cd0dfd2fdf92cb401192491963eb88ff0973a33fc5c7a3859c3c0d6a167c43d9` |
| move | `web/apps/realization/src/sale-offer-source.ts` | `web/apps/portal/src/features/realization/sale-offer-source.ts` | `8ed082138fb2fd785e780d1da829611a3268fd577590f2502e10ae53ee32b808` |
| move | `web/apps/realization/src/shared.tsx` | `web/apps/portal/src/features/realization/shared.tsx` | `41ce6cfc83f7cd711dc0e5f21db2cd2b527595da52d73e8af7180f051daafcb3` |
| move | `web/apps/realization/src/stock-colors.test.tsx` | `web/apps/portal/src/features/realization/stock-colors.test.tsx` | `a675ea64a1ac4bd8eaced5c9b7ceed000e6ed7f1fc48459810da85bbc2b39291` |
| move | `web/apps/realization/src/terms-dialog.test.tsx` | `web/apps/portal/src/features/realization/terms-dialog.test.tsx` | `753a082c2a69928c501f9c348c9714c6aed88117283c5913d8f00565e26573d3` |
| move | `web/apps/realization/src/vehicle-colors.test.ts` | `web/apps/portal/src/features/realization/vehicle-colors.test.ts` | `ae6d2e606027b1df92d6cbefb67931f39135479c742945c60d8375aebc382528` |
| move | `web/apps/realization/src/vehicle-colors.ts` | `web/apps/portal/src/features/realization/vehicle-colors.ts` | `b01fef3811fec5f243cc68d75d6a3af8208fa7b4ce459c5e76b3bfb851b2040d` |
| move | `web/apps/realization/src/vehicle-move.test.tsx` | `web/apps/portal/src/features/realization/vehicle-move.test.tsx` | `c1afe7ea31acea628a3fbca1ee8bb8a5cedcc4b2839a21bfd1f22110f9d63698` |
| move | `web/apps/realization/src/vehicle-move.tsx` | `web/apps/portal/src/features/realization/vehicle-move.tsx` | `7fe82e83df8025d98daa5a52906eb86472abdd5a8600d1b4d48411d5f160eba1` |
| move | `web/apps/realization/src/vin-editor.test.tsx` | `web/apps/portal/src/features/realization/vin-editor.test.tsx` | `c36cfdbaca89cc22bcd26a78d321f2536709f3acfa7a8b9c491ed8e63389691b` |
| move | `web/apps/realization/src/vin-editor.tsx` | `web/apps/portal/src/features/realization/vin-editor.tsx` | `c4de5a4f8374a5bc77931b169c93b954e71113d65bebf84c2fd9feb7ef28075f` |
| move | `web/apps/financing/src/design.css` | `web/apps/portal/src/features/financing/design.css` | `6318beeaefba4552a50e6087094311ef2e8c71bec49a82a118c5f26427922f7b` |
| replace-entry | `web/apps/financing/src/main.tsx` | `web/apps/portal/src/sections.tsx` (composition only) | `451d6e9d73dd9dc70ca6285bbe579906206b5473e2f7014b938efd106f628f43` |
| move | `web/apps/financing/src/navigation.tsx` | `web/apps/portal/src/features/financing/navigation.tsx` | `76af9a20e6393bb581ca25480ed3ee18c4c4bba1a37a8255c0cd14f50e8947c8` |
| move | `web/apps/financing/src/pages.tsx` | `web/apps/portal/src/features/financing/pages.tsx` | `013eda1398280bfd345d50cf8e21f6bec617d617803988854f6a0aaa77eec3d0` |
| move | `web/apps/financing/src/programs.test.tsx` | `web/apps/portal/src/features/financing/programs.test.tsx` | `8e96d897aa38cfb77faef2cba3b1d2ac7a71a71cdeb5e27c441c93b17f660d85` |
| move | `web/apps/financing/src/programs.ts` | `web/apps/portal/src/features/financing/programs.ts` | `d3015975375249490ad3344847f09453732312934000bce30d45606903b7c035` |
| replace-entry | `web/apps/insurance/src/main.tsx` | `web/apps/portal/src/sections.tsx` (composition only) | `c0075b2baebfa0f4417910b7dca8e9e2ad2d8810a6216dd8836affbbe5cab874` |
| move | `web/apps/insurance/src/navigation.tsx` | `web/apps/portal/src/features/insurance/navigation.tsx` | `a14f6ca7db83c78052f3723797c11878fc72d0f302d6bf71d0cb33b1f5e10e84` |
| move | `web/apps/insurance/src/pages.tsx` | `web/apps/portal/src/features/insurance/pages.tsx` | `5ae84ad053c7802fbf21085111e204ba8b62e0dd39efa893ac34f16f73b836f4` |
| move | `web/apps/insurance/src/queues.test.tsx` | `web/apps/portal/src/features/insurance/queues.test.tsx` | `28233083a8909eafe1458a4146ba754eb9f3298f360f27f5c41e8e501749d8b7` |
| move | `web/apps/insurance/src/queues.ts` | `web/apps/portal/src/features/insurance/queues.ts` | `072b3e5a7238ed57b86bd3db57aa863745e9efb32c83602d9c6141fa58c59bab` |
| move | `web/apps/admin/src/catalog-color-fields.tsx` | `web/apps/portal/src/features/admin/catalog-color-fields.tsx` | `2f3f2f561fcf7ec7e79ef56cd10b59824b2a47d2ffe112a68d3cb369cf061861` |
| move | `web/apps/admin/src/catalog-page.test.tsx` | `web/apps/portal/src/features/admin/catalog-page.test.tsx` | `239a170c326b9c4afd2df50e895e24a54b1da2709862bc99283eee5c3c580977` |
| move | `web/apps/admin/src/company-form.test.tsx` | `web/apps/portal/src/features/admin/company-form.test.tsx` | `ae089cb8982807de41739d179adb5420f578ca7f180243188873bee30cfac163` |
| move | `web/apps/admin/src/company-verification.test.tsx` | `web/apps/portal/src/features/admin/company-verification.test.tsx` | `dfdcfe9ed31e52a1e06a20f315a8613ba64f39518f2902923cd70e5a13a0a0a3` |
| move | `web/apps/admin/src/design.css` | `web/apps/portal/src/features/admin/design.css` | `167830db3415ed5e7958b619d2249ba0c94856ba7e2bf9febc50684a4f801522` |
| move | `web/apps/admin/src/labels.test.tsx` | `web/apps/portal/src/features/admin/labels.test.tsx` | `e8ee44184dfe7beecbb74a0fb34b9f84495f4d4c887ab96a186dc110f9e79d08` |
| move | `web/apps/admin/src/labels.ts` | `web/apps/portal/src/features/admin/labels.ts` | `8fdceb70d814b2934a1dde944741e9518122ba6820f6b42576996b57ebb3b75c` |
| replace-entry | `web/apps/admin/src/main.tsx` | `web/apps/portal/src/sections.tsx` (composition only) | `0d3cdac7fc204f19ac1378403eab9deb46cd0b28dbf9d683689658405e24c318` |
| move | `web/apps/admin/src/pages.tsx` | `web/apps/portal/src/features/admin/pages.tsx` | `0d97184f399a6228ba1936023734a12b17f607710e7abffb28152cce551ab924` |
| move | `web/apps/admin/src/permissions-page.test.tsx` | `web/apps/portal/src/features/admin/permissions-page.test.tsx` | `da21f817536667dd5c162d54143c5fe3a08d1014c1b907f1ed31c772ebb9225a` |
| move | `web/apps/admin/src/users-page.test.tsx` | `web/apps/portal/src/features/admin/users-page.test.tsx` | `bd6cc8d5ec219256857b7ef53e0edec8e0774aa55d58d4d6062fe421f4a363fc` |

