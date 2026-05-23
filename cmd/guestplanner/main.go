package main

import (
	"crypto/rand"
	_ "embed"
	"strings"
	"time"

	"github.com/torbenschinke/guestplanner/guest"
	"github.com/worldiety/enum"
	"github.com/worldiety/option"
	"go.wdy.de/nago/application"
	"go.wdy.de/nago/application/ent"
	cfgent "go.wdy.de/nago/application/ent/cfg"
	cfgflow "go.wdy.de/nago/application/flow/cfg"
	"go.wdy.de/nago/application/image"
	httpimage "go.wdy.de/nago/application/image/http"
	cfginspector "go.wdy.de/nago/application/inspector/cfg"
	"go.wdy.de/nago/application/settings"
	"go.wdy.de/nago/auth"
	"go.wdy.de/nago/pkg/xtime"
	"go.wdy.de/nago/presentation/core"
	icons "go.wdy.de/nago/presentation/icons/flowbite/outline"
	"go.wdy.de/nago/presentation/ui"
	"go.wdy.de/nago/presentation/ui/alert"
	"go.wdy.de/nago/presentation/ui/form"
	"go.wdy.de/nago/web/ssr/cfgssr"
	"go.wdy.de/nago/web/vuejs"
)

func main() {
	application.Configure(func(cfg *application.Configurator) {
		cfg.SetApplicationID("schinke.guestplanner")
		cfg.Serve(vuejs.Dist())

		option.MustZero(cfg.StandardSystems())
		modUser := option.Must(cfg.UserManagement())
		if option.Must(modUser.UseCases.CountUsers()) == 0 {
			option.Must(modUser.UseCases.EnableBootstrapAdmin(time.Now().Add(time.Hour), "%6UbRsdfg4dM8N$auy"))
		}

		cfg.SetDecorator(cfg.NewScaffold().Decorator())
		option.Must(cfginspector.Enable(cfg))
		option.Must(cfg.SettingsManagement())

		option.Must(cfgflow.Enable(cfg, cfgflow.Options{}))
		registerLink := cfg.ContextPathURI("", nil)

		modGuest := option.Must(cfgent.Enable[guest.Guest](cfg, "guestplanner.guest", "Gast", cfgent.Options[guest.Guest, guest.ID]{
			Searchable: true,

			DecorateUpdateView: func(wnd core.Window, state *core.State[guest.Guest], view core.View) core.View {
				cfg := core.GlobalSettings[GuestPlannerSettings](wnd)
				text := cfg.SinglePlannerInvitationText
				if !state.Get().Single {
					text = cfg.MultiPlannerInvitationText
				}

				text = strings.ReplaceAll(text, "$CODE", state.Get().RegistrationCode)
				text = strings.ReplaceAll(text, "$LINK", registerLink+"?code="+state.Get().RegistrationCode)
				text = strings.ReplaceAll(text, "$SALUTATION", state.Get().Salutation)
				return ui.VStack(
					view,
					form.Card(
						ui.H2("Einladung (SMS)"),
						ui.HStack(ui.SecondaryButton(func() {
							wnd.Clipboard().SetText(text)
						}).PreIcon(icons.Clipboard)).FullWidth().Alignment(ui.Trailing),
						ui.Text(text),
					),
				).FullWidth()
			},

			DecorateUseCases: func(uc ent.UseCases[guest.Guest, guest.ID]) ent.UseCases[guest.Guest, guest.ID] {

				fnCreate := uc.Create
				uc.Create = func(subject auth.Subject, entity guest.Guest) (id guest.ID, err error) {
					if entity.Status == "" {
						entity.Status = guest.StatusWaiting
					}

					if entity.RegistrationCode == "" {
						entity.RegistrationCode = rand.Text()[:6]
					}

					return fnCreate(subject, entity)
				}

				return uc
			},
		}))

		cfgssr.Enable(cfg, ".")

		cfg.RootView(".", func(wnd core.Window) core.View {
			cfg := core.GlobalSettings[GuestPlannerSettings](wnd)
			bannerUri := httpimage.URI(cfg.Banner, image.FitCover, 1920, 1080)
			code := wnd.Values()["code"]
			var mGuest guest.Guest
			for g, err := range modGuest.Repository.All() {
				if err != nil {
					return alert.BannerError(err)
				}

				if g.RegistrationCode == code {
					mGuest = g
					break
				}
			}

			if mGuest.ID == "" {
				return ui.VStack(
					ui.WindowTitle(cfg.GuestLandingPageTitle),
					ui.HStack().
						Background(ui.Background{}.AppendURI(bannerUri).Fit(ui.FitCover)).
						Frame(ui.Frame{Width: "100%", Height: "20rem", MaxWidth: "60rem"}),

					ui.VStack(
						ui.HStack(ui.Text(cfg.AnonWelcomeText)).BackgroundColor(ui.ColorWhite).
							Border(ui.Border{}.Radius(ui.L32).Shadow(ui.L16)).
							Padding(ui.Padding{}.All(ui.L32)).
							Frame(ui.Frame{Width: "100%", MaxWidth: "30rem"}),
					).NoClip(true).
						Padding(ui.Padding{Top: "-10rem"}),
				).FullWidth().NoClip(true)
			}

			text := cfg.SingleRegistrationText
			if !mGuest.Single {
				text = cfg.MultipleRegistrationText
			}

			text = strings.ReplaceAll(text, "$SALUTATION", mGuest.Salutation)
			registerDuration := cfg.Deadline.Time(time.Local).Sub(xtime.Now().Time(time.Local))
			canRegister := registerDuration > 0

			adults := core.AutoState[int64](wnd).Init(func() int64 {
				if mGuest.Single {
					return 1
				}

				return 2
			})

			children := core.AutoState[int64](wnd)

			var acceptTitle string
			var denyTitle string

			count := adults.Get() + children.Get()
			if count > 1 {
				acceptTitle = "Wir kommen"
				denyTitle = "Wir sind nicht dabei"
			} else {
				acceptTitle = "Ich komme"
				denyTitle = "Ich bin nicht dabei"
			}

			updateGuest := func(fn func(g guest.Guest) guest.Guest) {
				optG, err := modGuest.Repository.FindByID(mGuest.ID)
				if err != nil {
					alert.ShowBannerError(wnd, err)
					return
				}

				if optG.IsNone() {
					return
				}

				g := fn(optG.Unwrap())
				if err := modGuest.Repository.Save(g); err != nil {
					alert.ShowBannerError(wnd, err)
				}

				adults.Invalidate()
			}

			return ui.VStack(
				ui.HStack().
					Background(ui.Background{}.AppendURI(bannerUri).Fit(ui.FitCover)).
					Frame(ui.Frame{Width: "100%", Height: "20rem", MaxWidth: ui.L880}),

				ui.VStack(
					ui.VStack(
						ui.H1(cfg.RegistrationTitle),
						ui.HStack(
							ui.ImageIcon(icons.Clock),
							ui.Text(cfg.DateAndTime),
						).Gap(ui.L16),

						ui.HLine(),
						ui.HStack(
							ui.ImageIcon(icons.MapPin),
							ui.Text(cfg.Location),
						).Gap(ui.L16),

						ui.Space(ui.L32),
						ui.H2("Details"),
						ui.Text(text),

						ui.Space(ui.L32),
						ui.H2("Anmeldung"),
						func() core.View {
							if !canRegister || mGuest.Status != guest.StatusWaiting {
								var stateText string
								if !canRegister {
									stateText = "Leider ist der Anmeldeschluss bereits verstrichen. "
								}

								if mGuest.Status == guest.StatusConfirmed {
									if mGuest.Single {
										stateText += "Wir freuen uns, dass du dabei bist!"
									} else {
										stateText += "Wir freuen uns, dass ihr dabei seid!"
									}
								} else {
									if mGuest.Single {
										stateText += "Schade, dass du nicht dabei sein kannst."
									} else {
										stateText += "Schade, dass ihr nicht dabei sein könnt."
									}
								}

								return ui.VStack(
									ui.Text(stateText),
									ui.If(canRegister, ui.PrimaryButton(func() {
										updateGuest(func(g guest.Guest) guest.Guest {
											g.Status = guest.StatusWaiting
											return g
										})
									}).Title("Anmeldung ändern"),
									),
								).FullWidth().Gap(ui.L16)
							}

							return ui.VStack(

								ui.Text("Damit wir unsere Feier planen können, benötigen wir eine Anmeldung bis zum "+cfg.Deadline.Format("02.01.2006")+"."),

								ui.IfFunc(!mGuest.Single, func() core.View {
									return ui.VStack(
										ui.Space(ui.L32),
										ui.Text("Wir kommen mit"),
										ui.IntField("Erwachsenen", adults.Get(), adults).FullWidth(),
										ui.IntField("Kindern", children.Get(), children).FullWidth(),
									).Gap(ui.L16).FullWidth().Alignment(ui.Leading)
								}),

								ui.Space(ui.L32),
								ui.HStack(
									ui.SecondaryButton(func() {
										updateGuest(func(g guest.Guest) guest.Guest {
											g.Status = guest.StatusDeclined
											g.Adults = 0
											g.Children = 0
											return g
										})
									}).Title(denyTitle),

									ui.PrimaryButton(func() {
										updateGuest(func(g guest.Guest) guest.Guest {
											g.Status = guest.StatusConfirmed
											g.Adults = int(adults.Get())
											g.Children = int(children.Get())
											return g
										})
									}).Title(acceptTitle),
								).FullWidth().Gap(ui.L8),
							).FullWidth()
						}(),
					).BackgroundColor(ui.ColorWhite).
						FullWidth().
						Gap(ui.L16).
						Alignment(ui.Leading).
						Border(ui.Border{}.Radius(ui.L32).Shadow(ui.L16)).
						Padding(ui.Padding{}.All(ui.L32)),
				).NoClip(true).
					Padding(ui.Padding{Top: "-10rem"}).
					Frame(ui.Frame{Width: "100%", MaxWidth: ui.L560}),

				ui.Space(ui.L32),
			).FullWidth().NoClip(true)

		})

	}).Run()
}

var _ = enum.Variant[settings.GlobalSettings, GuestPlannerSettings]()

type GuestPlannerSettings struct {
	GuestLandingPageTitle    string
	Banner                   image.ID
	AnonWelcomeText          string     `lines:"5"`
	RegistrationTitle        string     `section:"Anmeldung Web" label:"Title"`
	DateAndTime              string     `section:"Anmeldung Web" label:"Wann"`
	Location                 string     `section:"Anmeldung Web" label:"Wo"`
	Deadline                 xtime.Date `section:"Anmeldung Web" label:"Anmeldeschluss"`
	SingleRegistrationText   string     `section:"Anmeldung Web" lines:"5" label:"Text Einzelanmeldung" supportingText:"Unterstützt wird der Platzhalter $SALUTATION für die Anrede."`
	MultipleRegistrationText string     `section:"Anmeldung Web" lines:"5" label:"Text Mehrfachanmeldung"`

	SinglePlannerInvitationText string `section:"Einladung (SMS)" label:"Text Einladung Einzel" lines:"5" supportingText:"Unterstützt werden die Platzhalter $CODE für den Registrierungscode, $LINK für den Registrierungslink und $SALUTATION für die Anrede."`
	MultiPlannerInvitationText  string `section:"Einladung (SMS)" label:"Text Einladung Mehrfach" lines:"5" `
}

func (g GuestPlannerSettings) GlobalSettings() bool {
	return true
}
