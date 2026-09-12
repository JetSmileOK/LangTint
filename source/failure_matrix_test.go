package main

import (
	"fmt"
	"testing"
)

type failureScenario struct {
	name string
	run  func(t *testing.T)
}

func TestFailureMatrix100DistinctScenarios(t *testing.T) {
	var scenarios []failureScenario
	add := func(name string, fn func(t *testing.T)) {
		scenarios = append(scenarios, failureScenario{name: name, run: fn})
	}

	// 01-15: OS compatibility boundaries and future/legacy failure modes.
	osCases := []struct {
		name                string
		major, minor, build uint32
		want                bool
	}{
		{"01_win7_rejected", 6, 1, 7601, false},
		{"02_win81_rejected", 6, 3, 9600, false},
		{"03_win10_1507_rejected", 10, 0, 10240, false},
		{"04_win10_1511_rejected", 10, 0, 10586, false},
		{"05_one_build_before_1607_rejected", 10, 0, 14392, false},
		{"06_win10_1607_minimum", 10, 0, 14393, true},
		{"07_win10_1703", 10, 0, 15063, true},
		{"08_win10_1809", 10, 0, 17763, true},
		{"09_win10_1903", 10, 0, 18362, true},
		{"10_win10_2004", 10, 0, 19041, true},
		{"11_win10_22h2", 10, 0, 19045, true},
		{"12_last_pre_win11_build", 10, 0, 21999, true},
		{"13_win11_22000_rejected", 10, 0, 22000, false},
		{"14_win11_23h2_rejected", 10, 0, 22631, false},
		{"15_future_major_rejected", 11, 0, 30000, false},
	}
	for _, tc := range osCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			if got := windowsBuildSupport(tc.major, tc.minor, tc.build).Supported; got != tc.want {
				t.Fatalf("supported=%v want=%v", got, tc.want)
			}
		})
	}

	// 16-25: verify the taskbar owner path cannot be spoofed by class name alone.
	pathCases := []struct {
		name, path, winDir string
		want               bool
	}{
		{"16_real_explorer_default_drive", `C:\Windows\explorer.exe`, `C:\Windows`, true},
		{"17_real_explorer_case_insensitive", `c:\windows\EXPLORER.EXE`, `C:\Windows`, true},
		{"18_real_explorer_forward_slashes", `C:/Windows/explorer.exe`, `C:\Windows`, true},
		{"19_real_explorer_alt_windows_drive", `D:\Windows\explorer.exe`, `D:\Windows`, true},
		{"20_fake_explorer_temp_rejected", `C:\Temp\explorer.exe`, `C:\Windows`, false},
		{"21_fake_explorer_user_rejected", `C:\Users\A\explorer.exe`, `C:\Windows`, false},
		{"22_similar_explorer_name_rejected", `C:\Windows\myexplorer.exe`, `C:\Windows`, false},
		{"23_explorer_directory_rejected", `C:\Windows\explorer.exe\child`, `C:\Windows`, false},
		{"24_empty_process_path_rejected", ``, `C:\Windows`, false},
		{"25_empty_windows_dir_rejected", `C:\Windows\explorer.exe`, ``, false},
	}
	for _, tc := range pathCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			if got := isExpectedExplorerProcessPath(tc.path, tc.winDir); got != tc.want {
				t.Fatalf("got=%v want=%v path=%q winDir=%q", got, tc.want, tc.path, tc.winDir)
			}
		})
	}

	// 26-50: localized MSAA names. These simulate Windows UI languages without
	// hardcoding English/Russian words into the runtime decision path.
	localizedCases := []struct {
		name, text, candidate string
		lang                  Language
	}{
		{"26_ru_ui_english", "Индикатор ввода - Английский (США)", "Английский (США)", LanguageEnglish},
		{"27_de_ui_english", "Eingabeindikator - Englisch (Vereinigte Staaten)", "Englisch (Vereinigte Staaten)", LanguageEnglish},
		{"28_fr_ui_english", "Indicateur de saisie - Anglais (États-Unis)", "Anglais (États-Unis)", LanguageEnglish},
		{"29_es_ui_english", "Indicador de entrada - Inglés (Estados Unidos)", "Inglés (Estados Unidos)", LanguageEnglish},
		{"30_it_ui_english", "Indicatore di input - Inglese (Stati Uniti)", "Inglese (Stati Uniti)", LanguageEnglish},
		{"31_pl_ui_english", "Wskaźnik wprowadzania - Angielski (Stany Zjednoczone)", "Angielski (Stany Zjednoczone)", LanguageEnglish},
		{"32_uk_ui_english", "Індикатор вводу - Англійська (Сполучені Штати)", "Англійська (Сполучені Штати)", LanguageEnglish},
		{"33_ja_ui_english", "入力インジケーター - 英語 (米国)", "英語 (米国)", LanguageEnglish},
		{"34_ko_ui_english", "입력 표시기 - 영어 (미국)", "영어 (미국)", LanguageEnglish},
		{"35_tr_ui_english", "Giriş göstergesi - İngilizce (ABD)", "İngilizce (ABD)", LanguageEnglish},
		{"36_nl_ui_english", "Invoerindicator - Engels (Verenigde Staten)", "Engels (Verenigde Staten)", LanguageEnglish},
		{"37_pt_ui_english", "Indicador de entrada - Inglês (Estados Unidos)", "Inglês (Estados Unidos)", LanguageEnglish},
		{"38_ro_ui_english", "Indicator de intrare - Engleză (Statele Unite)", "Engleză (Statele Unite)", LanguageEnglish},
		{"39_hu_ui_english", "Beviteli jelző - angol (Egyesült Államok)", "angol (Egyesült Államok)", LanguageEnglish},
		{"40_nbsp_and_spacing", "Input\u00a0 indicator   -   English (United States)", "English (United States)", LanguageEnglish},
		{"41_de_ui_russian", "Eingabeindikator - Russisch", "Russisch", LanguageRussian},
		{"42_fr_ui_russian", "Indicateur de saisie - Russe", "Russe", LanguageRussian},
		{"43_es_ui_russian", "Indicador de entrada - Ruso", "Ruso", LanguageRussian},
		{"44_pl_ui_russian", "Wskaźnik wprowadzania - Rosyjski", "Rosyjski", LanguageRussian},
		{"45_ja_ui_russian", "入力インジケーター - ロシア語", "ロシア語", LanguageRussian},
		{"46_de_other_language", "Eingabeindikator - Deutsch (Deutschland)", "Deutsch (Deutschland)", LanguageOther},
		{"47_fr_other_language", "Indicateur de saisie - Français (France)", "Français (France)", LanguageOther},
		{"48_uk_other_language", "Індикатор вводу - Українська", "Українська", LanguageOther},
		{"49_japanese_other_language", "入力インジケーター - 日本語", "日本語", LanguageOther},
		{"50_unmatched_name_fails_safe", "Input indicator - Klingon", "English", LanguageUnknown},
	}
	for _, tc := range localizedCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			candidateLang := tc.lang
			if tc.lang == LanguageUnknown {
				candidateLang = LanguageEnglish
			}
			got := classifyAccessibleNameWithCandidates(tc.text, []languageNameCandidate{{Name: tc.candidate, Language: candidateLang}})
			if got != tc.lang {
				t.Fatalf("got=%v want=%v", got, tc.lang)
			}
		})
	}

	// 51-65: HKL language classification, including many English variants.
	hklCases := []struct {
		name string
		hkl  uintptr
		want Language
	}{
		{"51_hkl_zero_unknown", 0, LanguageUnknown},
		{"52_en_us", 0x0409, LanguageEnglish},
		{"53_en_gb", 0x0809, LanguageEnglish},
		{"54_en_au", 0x0c09, LanguageEnglish},
		{"55_en_ca", 0x1009, LanguageEnglish},
		{"56_en_nz", 0x1409, LanguageEnglish},
		{"57_en_ie", 0x1809, LanguageEnglish},
		{"58_en_za", 0x1c09, LanguageEnglish},
		{"59_ru_ru", 0x0419, LanguageRussian},
		{"60_de_de_other", 0x0407, LanguageOther},
		{"61_fr_fr_other", 0x040c, LanguageOther},
		{"62_es_es_other", 0x0c0a, LanguageOther},
		{"63_uk_ua_other", 0x0422, LanguageOther},
		{"64_pl_pl_other", 0x0415, LanguageOther},
		{"65_ja_jp_other", 0x0411, LanguageOther},
	}
	for _, tc := range hklCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			if got := languageFromHKL(tc.hkl); got != tc.want {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
		})
	}

	// 66-85: distinct keyboard event failure/edge scenarios.
	hotkeyCases := []struct {
		name string
		seq  []struct {
			vk   uint32
			down bool
		}
		want int
	}{
		{"66_left_alt_left_shift", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), kd(vkLShift), ku(vkLShift), ku(vkLMenu)}, 1},
		{"67_right_alt_right_shift", []struct {
			vk   uint32
			down bool
		}{kd(vkRMenu), kd(vkRShift), ku(vkRShift), ku(vkRMenu)}, 1},
		{"68_left_alt_right_shift", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), kd(vkRShift), ku(vkRShift), ku(vkLMenu)}, 1},
		{"69_shift_before_alt", []struct {
			vk   uint32
			down bool
		}{kd(vkLShift), kd(vkLMenu), ku(vkLMenu), ku(vkLShift)}, 1},
		{"70_left_ctrl_shift", []struct {
			vk   uint32
			down bool
		}{kd(vkLControl), kd(vkLShift), ku(vkLShift), ku(vkLControl)}, 1},
		{"71_right_ctrl_shift", []struct {
			vk   uint32
			down bool
		}{kd(vkRControl), kd(vkRShift), ku(vkRShift), ku(vkRControl)}, 1},
		{"72_left_win_space", []struct {
			vk   uint32
			down bool
		}{kd(vkLWin), kd(vkSpace), ku(vkSpace), ku(vkLWin)}, 1},
		{"73_right_win_space", []struct {
			vk   uint32
			down bool
		}{kd(vkRWin), kd(vkSpace), ku(vkSpace), ku(vkRWin)}, 1},
		{"74_plain_letter", []struct {
			vk   uint32
			down bool
		}{kd('A'), ku('A')}, 0},
		{"75_alt_only", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), ku(vkLMenu)}, 0},
		{"76_shift_only", []struct {
			vk   uint32
			down bool
		}{kd(vkLShift), ku(vkLShift)}, 0},
		{"77_ctrl_only", []struct {
			vk   uint32
			down bool
		}{kd(vkLControl), ku(vkLControl)}, 0},
		{"78_win_only", []struct {
			vk   uint32
			down bool
		}{kd(vkLWin), ku(vkLWin)}, 0},
		{"79_space_only", []struct {
			vk   uint32
			down bool
		}{kd(vkSpace), ku(vkSpace)}, 0},
		{"80_altgr_typing", []struct {
			vk   uint32
			down bool
		}{kd(vkRMenu), kd(vkLControl), kd('Q'), ku('Q'), ku(vkLControl), ku(vkRMenu)}, 0},
		{"81_shift_autorepeat_held", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), kd(vkLShift), kd(vkLShift), kd(vkLShift), ku(vkLShift), ku(vkLMenu)}, 1},
		{"82_noise_inside_chord", []struct {
			vk   uint32
			down bool
		}{kd('X'), ku('X'), kd(vkLMenu), kd(vkLShift), kd('A'), ku('A'), ku(vkLShift), ku(vkLMenu)}, 1},
		{"83_two_altshift_cycles", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), kd(vkLShift), ku(vkLShift), ku(vkLMenu), kd(vkLMenu), kd(vkLShift), ku(vkLShift), ku(vkLMenu)}, 2},
		{"84_ctrl_shift_then_win_space", []struct {
			vk   uint32
			down bool
		}{kd(vkLControl), kd(vkLShift), ku(vkLShift), ku(vkLControl), kd(vkLWin), kd(vkSpace), ku(vkSpace), ku(vkLWin)}, 2},
		{"85_incomplete_chord_no_release", []struct {
			vk   uint32
			down bool
		}{kd(vkLMenu), kd(vkLShift), ku(vkLShift)}, 0},
	}
	for _, tc := range hotkeyCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			d := &HotkeyDetector{}
			if got := feed(d, tc.seq...); got != tc.want {
				t.Fatalf("got=%d want=%d", got, tc.want)
			}
		})
	}

	// 86-95: DWM visual coalescing/fail-safe state behavior.
	visualCases := []struct {
		name  string
		steps []struct{ tint, force, want bool }
		reset bool
	}{
		{"86_initial_ru_applies", []struct{ tint, force, want bool }{{false, false, true}}, false},
		{"87_initial_en_applies", []struct{ tint, force, want bool }{{true, false, true}}, false},
		{"88_duplicate_ru_suppressed", []struct{ tint, force, want bool }{{false, false, true}, {false, false, false}}, false},
		{"89_duplicate_en_suppressed", []struct{ tint, force, want bool }{{true, false, true}, {true, false, false}}, false},
		{"90_ru_to_en_applies", []struct{ tint, force, want bool }{{false, false, true}, {true, false, true}}, false},
		{"91_en_to_ru_applies", []struct{ tint, force, want bool }{{true, false, true}, {false, false, true}}, false},
		{"92_force_en_rebind_applies", []struct{ tint, force, want bool }{{true, false, true}, {true, true, true}}, false},
		{"93_force_ru_rebind_applies", []struct{ tint, force, want bool }{{false, false, true}, {false, true, true}}, false},
		{"94_reset_forces_next_apply", []struct{ tint, force, want bool }{{true, false, true}, {true, false, false}}, true},
		{"95_multiple_transitions", []struct{ tint, force, want bool }{{false, false, true}, {true, false, true}, {false, false, true}, {true, false, true}}, false},
	}
	for _, tc := range visualCases {
		tc := tc
		add(tc.name, func(t *testing.T) {
			var g VisualGate
			for i, step := range tc.steps {
				if got := g.Decide(step.tint, step.force); got != step.want {
					t.Fatalf("step=%d got=%v want=%v", i, got, step.want)
				}
			}
			if tc.reset {
				g.Reset()
				if !g.Decide(true, false) {
					t.Fatal("reset did not make next state observable")
				}
			}
		})
	}

	// 96-100: installer/startup safety invariants.
	add("96_no_args_is_safe_product_launch", func(t *testing.T) {
		if got := resolveMode(nil); got != "--launch" {
			t.Fatalf("got=%q", got)
		}
	})
	add("97_explicit_run_is_only_watcher_mode", func(t *testing.T) {
		if got := resolveMode([]string{"--run"}); got != "--run" {
			t.Fatalf("got=%q", got)
		}
	})
	add("98_unknown_argument_rejected", func(t *testing.T) {
		if got := resolveMode([]string{"--definitely-invalid"}); got != "--invalid" {
			t.Fatalf("got=%q", got)
		}
	})
	add("99_startup_retry_is_bounded", func(t *testing.T) {
		if got := startupRetryTotalMS(); got != 31750 {
			t.Fatalf("total=%d want=31750", got)
		}
	})
	add("100_startup_retry_has_seven_distinct_backoff_steps", func(t *testing.T) {
		if len(startupRetryDelaysMS) != 7 {
			t.Fatalf("len=%d", len(startupRetryDelaysMS))
		}
		for i := 1; i < len(startupRetryDelaysMS); i++ {
			if startupRetryDelaysMS[i] <= startupRetryDelaysMS[i-1] {
				t.Fatalf("delay[%d]=%d not greater than previous=%d", i, startupRetryDelaysMS[i], startupRetryDelaysMS[i-1])
			}
		}
	})

	if len(scenarios) != 100 {
		t.Fatalf("internal test design error: scenario count=%d want=100", len(scenarios))
	}
	seen := make(map[string]bool, len(scenarios))
	for i, sc := range scenarios {
		if seen[sc.name] {
			t.Fatalf("duplicate scenario name %q", sc.name)
		}
		seen[sc.name] = true
		t.Run(fmt.Sprintf("%03d_%s", i+1, sc.name), sc.run)
	}
}
