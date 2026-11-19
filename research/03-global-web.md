# 리서치 03 — 해외 커뮤니티·웹 (해커뉴스 직접 검색 + 웹 검색)

수집일: 2026-11-19

## 수집 방법과 한계

- **직접 읽은 것**: 해커뉴스(HN) 공개 검색 API로 2025년 이후 글을 검색하고, 댓글이 많은 스레드 9개를 열어 읽었다. 아래 인용은 원문 그대로다.
- **검색 요약으로만 확인한 것**: Reddit, Indie Hackers, dev.to, Product Hunt, 각종 블로그는 이 세션에서 접속이 막혀 있어 웹 검색 결과의 요약만 봤다. 해당 항목은 "검색 요약"으로 표시했고 수치는 검증되지 않았다.
- Indie Hackers는 자동 접근을 차단해 원문을 읽지 못했다.
- HN은 개발자 중심 커뮤니티라 비개발자 초보의 목소리는 덜 잡힌다. 이 편향은 한국 웹·유튜브 수집으로 보완한다.

## 단계별 페인포인트 (원문 인용)

### 4 홍보·첫 사용자 — 가장 자주, 가장 강하게 나온다

- 같은 패턴의 반복: "I've shipped multiple products over the past few years. Every single one followed the same pattern: build, post, get 12 likes from friends, a bit of organic traction, then nothing. Back to coding a new thing." — Ask HN: How do you handle marketing as a solo technical founder? (147점, 댓글 112, 2026-04) https://news.ycombinator.com/item?id=47667504
- 콜드 스타트: "every distribution channel has a cold start problem. you need traction to get traction. Even posting about your work requires reputation you can only build by posting." — Ask HN: How did you get your first users with zero audience? (2026-04) https://news.ycombinator.com/item?id=47800507
- 방문은 오는데 아무도 안 쓴다: "I built products, posted on Reddit and had literally thousands of people come to my site. Not one person bothered to go…" — https://news.ycombinator.com/item?id=47669239
- 정답이 없다는 체념: "I think the reality is no one really knows… to me it seems like the lottery." — https://news.ycombinator.com/item?id=47801740
- 압도감: "I really enjoy building but when it comes to distribution I feel overwhelmed." — https://news.ycombinator.com/item?id=47825351
- AI 시대의 소음: "the market just seems so heavily saturated with new products, that it's incredibly hard to break through the constant drone of people vying for attention" — Ask HN: How do you do marketing in the age of slop? (2026-07) https://news.ycombinator.com/item?id=48918150
- 광고는 클릭만 온다: "paid search ads etc have resulted in lots of clicks but absolutely abhorrent conversions in our case, whereas conversions for the more organic channels like forums etc are excellent" — 같은 스레드 (https://news.ycombinator.com/item?id=48918941 에 인용됨)

### 5 검증 — 만들고 나서야 검증을 고민한다

- "got a hard stop, impostor syndrome, at this step of having it almost ready and looking in the wild without knowing how to find if this is really valuable to someone else than me." — Ask HN: How do you manage distribution after building? (2026-06) https://news.ycombinator.com/item?id=48430217
- 경험자의 반응은 한결같다: "You seem to have missed the key step. Talk to customers before you build." — https://news.ycombinator.com/item?id=47668158
- "If you have no one to launch to, who are you building for? How did you validate your idea?" — https://news.ycombinator.com/item?id=47803603
- "Who did you talk to to validate the product design when developing your MVP? They are your first users." — https://news.ycombinator.com/item?id=49885462

### 0 아이템·문제정의 — 고객을 한 문장으로 못 쓰면 채널도 못 찾는다

- "If you describe your MVP using the following pattern: [customer profile] has [problem] so I built [solution] Then you should have an idea of who the product is for and then you can figure out where to find them." — https://news.ycombinator.com/item?id=49884408

### 2 제작·품질 — 빨리 만든 것은 빨리 죽는다

- SaaSHub 집계를 인용한 글: 제품 수명 중앙값이 약 4년에서 약 1.5년으로 줄었다는 주장. "A lot of products are being vibe coded, released, and soon after that abandoned." (92점, 2026-10) https://news.ycombinator.com/item?id=50204125 — 집계 방식은 확인하지 못했고, 댓글에서 집계 사이트 자체의 품질을 의심하는 반론도 있었다.
- Show HN에 올라온 바이브코딩 앱 155개를 스캔한 글의 댓글: "36% of launched platform-hosted links are already dead (404)" (2026-10) https://news.ycombinator.com/item?id=50133249 — 개인이 한 소규모 조사다.
- 보안 스캐너·체크리스트 Show HN이 2025~2026년에 십수 건 올라왔다(검색 결과 51건). 보안은 이미 도구가 많이 나온 영역이다.

### 8 운영·지속 — 버티기와 학습 손실

- "Most launches are received with dead silence. My favorite definition of a startup is: 'a group of people trying to survive long enough to get lucky'." — https://news.ycombinator.com/item?id=47825351
- 싸게 만들고 싸게 버리면 배우는 게 없다는 경고: "you launch a thing and nobody gives a shit especially you yourself… you risk learning nothing at all" — https://news.ycombinator.com/item?id=50204421
- "Marketing is a skill. But you won't get good at it unless you try and fail many, many times." — https://news.ycombinator.com/item?id=43662825

## 효과가 있다고 반복해서 언급되는 방법

여러 스레드에서 독립적으로 반복된 순서대로 적었다. 개인 경험담이며 통계가 아니다.

1. **만들기 전에 사람과 대화한다.** 팔려고 하지 말고 그 사람의 문제를 묻는다. 대화한 사람이 곧 첫 사용자가 된다.
2. **고객이 모이는 곳을 2~3곳으로 좁힌다.** 창업자들이 모이는 곳이 아니라 고객이 질문하는 곳이어야 한다. "Spend 20 minutes a day answering those questions thoroughly before mentioning what you build." (https://news.ycombinator.com/item?id=48920536)
3. **첫 100명은 한 명씩 만난다.** "First 100 users, meet them one by one wherever you can (forums, friends, ex-coworkers)… Next 500 will most likely come from referrals" (https://news.ycombinator.com/item?id=43971134) / "Start small!! Set a short goal of 10-20 users"
4. **제품이 아니라 문제에 대해 쓴다.** "writing about the problem i was solving, not the product… people engage with the take, then they find the tool naturally." (https://news.ycombinator.com/item?id=47669781)
5. **검색으로 발견되는 구조를 만든다(장기).** 가입 없이 바로 써보는 랜딩, 작은 무료 도구·계산기로 유입을 만드는 "engineering as marketing". (https://news.ycombinator.com/item?id=47668654, https://news.ycombinator.com/item?id=43971216) 최근에는 AI 답변에 노출되는 것(GEO)이 유일한 유입원이라는 사례도 나왔다. (https://news.ycombinator.com/item?id=49890270)
6. **고객 한 명 한 명에게 연락해 이유를 묻고 반영한다.** "Make contact with every customer you acquire, find out why they installed your product" (https://news.ycombinator.com/item?id=47669134)

### 반복해서 지목되는 안티패턴

- 검증 없이 만들고 나서 홍보로 해결하려는 것
- 커뮤니티에 가입하자마자 자기 홍보만 하는 것 ("No one likes someone who joins a community just to self-promote.")
- 제품 검증 전에 유료 광고부터 집행하는 것 (검색 요약: 17달러 고객 한 명에 5,000달러를 쓴 사례가 Reddit 기반 글에서 인용됨, 미검증)
- 반응이 없으면 기능을 더 만들거나 다음 제품으로 넘어가는 것
- Product Hunt 런칭을 전략으로 착각하는 것 (검색 요약)

## 확인 가능한 데이터

- 1차 출처를 직접 확인한 정량 데이터는 이번 수집에서 확보하지 못했다.
- 자주 인용되는 "CB Insights: 실패 스타트업의 42%는 시장 수요 부재"는 여러 글이 재인용하고 있으나 원문을 열어 확인하지 않았다. 가이드에 쓰려면 원문 확인이 필요하다.
- 보안 스캔 수치(예: Supabase 기반 앱 다수가 공개 읽기 상태)는 보안 업체 블로그 발이라 표본과 정의가 제각각이다. "보안 점검은 필수"라는 방향성의 근거로만 쓴다.

## 경쟁 제품 지도와 공백 (검색 요약 기반)

| 유형 | 예 | 다루는 단계 | 형식 | 알려진 한계 |
|---|---|---|---|---|
| AI 코파운더 | Buildpad | 문제 발굴 → 검증 → 경쟁 분석 → 개발 계획 (7단계) | 단계형 채팅, 프리미엄(무료 일 10크레딧) | 리뷰 대부분이 홍보성. 검증의 질에 대한 독립 평가는 찾지 못함 |
| 아이디어 검증기 | ValidatorAI | 아이디어 즉석 피드백 | 채팅 | "AI opinion is not market evidence" — AI의 의견이지 시장 증거가 아니다 |
| 아이디어 DB | BigIdeasDB, IdeaBrowser | 아이템 탐색 | 데이터베이스 | 탐색에서 끝남 |
| 보일러플레이트 | ShipFast 류 | 제작·결제 연동 | 코드 템플릿 | 만들기 이후는 다루지 않음 |
| 보안 스캐너 | Show HN에 다수 | 제작(보안) | 자동 점검 | 이미 과밀 |
| 배포 계획 도구 | z2one 등 | 홍보 계획·추적 | 플래너 | 소규모, 평가 자료 없음 |

정리:

- **과밀**: 제작(빌더, 보일러플레이트, 튜토리얼), 보안 스캔, AI 아이디어 평가.
- **빈자리**: (1) AI의 의견이 아니라 실제 사람의 반응을 받아 오게 만드는 검증 실행, (2) 고객이 있는 곳을 찾고 규칙을 지키며 매일 조금씩 하는 홍보 실행, (3) 단계 사이를 잇는 판단 기준 — 언제 더 만들고, 언제 방향을 바꾸고, 언제 접는가.
- 세션 초기에 세운 가설("제작은 과밀, 검증·홍보 실행은 빈약")은 HN 스레드와 검색 요약 범위에서는 지지됐다. 단, 영어권 도구를 직접 써보고 확인한 것은 아니다.

## 회의·반발

- 구루·강의 판매에 대한 피로, 성공담의 생존자 편향, AI로 찍어낸 제품(slop)에 대한 거부감이 뚜렷하다.
- 사용자 쪽도 변했다: "I'm far less interested in adopting new products… amidst a sea of half-baked and short-lived products" (https://news.ycombinator.com/item?id=50204844). 신생 서비스는 "금방 사라질 것"이라는 의심을 먼저 넘어야 한다.
- 가이드가 지켜야 할 것: 수익을 약속하지 않는다, 실패 확률을 숨기지 않는다, AI 평가를 시장 증거처럼 포장하지 않는다.

## 웹 검색으로 찾은 참고 글 (원문 미확인)

- https://vantaige.io/blog/vibe-coding-no-customers-ai-marketing-fix-2026
- https://www.producthunt.com/p/murror/you-shipped-it-in-a-weekend-that-s-exactly-why-nobody-s-using-it
- https://dev.to/alichherawalla/why-your-vibe-coded-app-has-users-but-no-revenue-ka8
- https://dev.to/mrclaw207/your-vibe-coded-side-hustle-has-no-customers-and-its-not-because-of-the-code-4217
- https://www.indiehackers.com/post/3-products-live-0-paying-customers-heres-what-i-ve-learned-about-distribution-in-30-days-552dc73251
- https://analyticsindiamag.com/ai-features/vibe-coding-lets-anyone-start-but-few-know-how-to-finish
- https://redwerk.com/blog/vibe-code-maintenance/
- https://aicofounder.com/blog/buildpad-reviews-what-it-is-and-what-founders-are-saying-about-it-in-2026
